package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*destinationResource)(nil)
	_ resource.ResourceWithConfigure   = (*destinationResource)(nil)
	_ resource.ResourceWithImportState = (*destinationResource)(nil)
)

// NewDestinationResource is registered in provider.go.
func NewDestinationResource() resource.Resource {
	return &destinationResource{}
}

type destinationResource struct {
	client *client.Client
}

type destinationResourceModel struct {
	UUID       types.String `tfsdk:"uuid"`
	ServerUUID types.String `tfsdk:"server_uuid"`
	Name       types.String `tfsdk:"name"`
	Network    types.String `tfsdk:"network"`
	Type       types.String `tfsdk:"type"`
}

func (r *destinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (r *destinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Docker network (destination) on a Coolify server. Resources deploy " +
			"into a destination; servers get a default one when validated.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the server hosting the network. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name. Defaults to `<server>-<network>`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "Docker network name. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "`standalone` or `swarm`. Must match the server type; " +
					"defaults to the server's expected type.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("standalone", "swarm")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *destinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *destinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan destinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dest, err := r.client.CreateDestination(ctx, plan.ServerUUID.ValueString(), client.DestinationCreateRequest{
		Name:    stringOrNil(plan.Name),
		Network: stringOrNil(plan.Network),
		Type:    stringOrNil(plan.Type),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify destination", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, destinationToModel(dest, plan))...)
}

func (r *destinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state destinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dest, err := r.client.GetDestination(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify destination %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, destinationToModel(dest, state))...)
}

func (r *destinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan destinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dest, err := r.client.UpdateDestination(ctx, plan.UUID.ValueString(), client.DestinationUpdateRequest{
		Name: stringOrNil(plan.Name),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify destination %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, destinationToModel(dest, plan))...)
}

func (r *destinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state destinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteDestination(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify destination %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

func (r *destinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func destinationToModel(d *client.Destination, prior destinationResourceModel) destinationResourceModel {
	m := destinationResourceModel{
		UUID:       types.StringValue(d.UUID),
		Name:       types.StringValue(d.Name),
		Network:    types.StringValue(d.Network),
		ServerUUID: prior.ServerUUID,
		Type:       prior.Type,
	}
	// On import the prior state has no server reference; take it from the API.
	if m.ServerUUID.IsNull() || m.ServerUUID.IsUnknown() {
		if s := d.ServerRef(); s != "" {
			m.ServerUUID = types.StringValue(s)
		}
	}
	// The list/show endpoints do not echo the type back explicitly; keep the
	// planned/known value, defaulting to standalone for fresh imports.
	if m.Type.IsNull() || m.Type.IsUnknown() {
		m.Type = types.StringValue("standalone")
	}
	return m
}
