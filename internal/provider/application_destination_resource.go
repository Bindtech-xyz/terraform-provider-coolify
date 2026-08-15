package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*applicationDestinationResource)(nil)
	_ resource.ResourceWithConfigure   = (*applicationDestinationResource)(nil)
	_ resource.ResourceWithImportState = (*applicationDestinationResource)(nil)
)

// NewApplicationDestinationResource is registered in provider.go.
func NewApplicationDestinationResource() resource.Resource {
	return &applicationDestinationResource{}
}

type applicationDestinationResource struct {
	client *client.Client
}

type applicationDestinationResourceModel struct {
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	DestinationUUID types.String `tfsdk:"destination_uuid"`
}

func (r *applicationDestinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_destination"
}

func (r *applicationDestinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches an *additional* standalone Docker destination to an " +
			"application (multi-destination deployment) — not for the primary destination, which " +
			"is `destination_uuid` on `coolify_application` itself. Coolify rejects a destination " +
			"already on the same server as the primary or another attached destination. Requires " +
			"Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the application. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"destination_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the additional destination to attach. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *applicationDestinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *applicationDestinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationDestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AddApplicationDestination(ctx, plan.ApplicationUUID.ValueString(), plan.DestinationUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to attach Coolify application destination", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *applicationDestinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationDestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dests, err := r.client.ListApplicationDestinations(ctx, state.ApplicationUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read destinations of application %s", state.ApplicationUUID.ValueString()),
			err.Error(),
		)
		return
	}
	found := false
	for _, d := range dests {
		if d.UUID == state.DestinationUUID.ValueString() && !d.IsPrimary {
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update never runs: every attribute forces replacement.
func (r *applicationDestinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan applicationDestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *applicationDestinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationDestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemoveApplicationDestination(ctx, state.ApplicationUUID.ValueString(), state.DestinationUUID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to detach Coolify application destination %s", state.DestinationUUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects "<application_uuid>/<destination_uuid>".
func (r *applicationDestinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<application_uuid>/<destination_uuid>\", got %q.", req.ID),
		)
		return
	}

	state := applicationDestinationResourceModel{
		ApplicationUUID: types.StringValue(parts[0]),
		DestinationUUID: types.StringValue(parts[1]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	// Read() runs automatically after import to confirm existence.
}
