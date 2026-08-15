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

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*cloudTokenResource)(nil)
	_ resource.ResourceWithConfigure   = (*cloudTokenResource)(nil)
	_ resource.ResourceWithImportState = (*cloudTokenResource)(nil)
)

// NewCloudTokenResource is registered in provider.go.
func NewCloudTokenResource() resource.Resource {
	return &cloudTokenResource{}
}

type cloudTokenResource struct {
	client *client.Client
}

type cloudTokenResourceModel struct {
	UUID     types.String `tfsdk:"uuid"`
	Name     types.String `tfsdk:"name"`
	Provider types.String `tfsdk:"provider_name"`
	Token    types.String `tfsdk:"token"`
}

func (r *cloudTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_token"
}

func (r *cloudTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An API token for a cloud provider (Hetzner, DigitalOcean or Vultr), " +
			"used by `coolify_cloud_server` to provision VMs.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name.",
				Required:            true,
			},
			"provider_name": schema.StringAttribute{
				MarkdownDescription: "`hetzner`, `digitalocean` or `vultr`. Changing it forces replacement.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(client.CloudProviders...)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The provider API token.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *cloudTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *cloudTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.CreateCloudToken(ctx, client.CloudTokenRequest{
		Name:     stringOrNil(plan.Name),
		Provider: stringOrNil(plan.Provider),
		Token:    stringOrNil(plan.Token),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify cloud token", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudTokenToModel(token, plan))...)
}

func (r *cloudTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.GetCloudToken(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify cloud token %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudTokenToModel(token, state))...)
}

func (r *cloudTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.UpdateCloudToken(ctx, plan.UUID.ValueString(), client.CloudTokenRequest{
		Name:  stringOrNil(plan.Name),
		Token: stringOrNil(plan.Token),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify cloud token %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudTokenToModel(token, plan))...)
}

func (r *cloudTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteCloudToken(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify cloud token %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

func (r *cloudTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func cloudTokenToModel(t *client.CloudToken, prior cloudTokenResourceModel) cloudTokenResourceModel {
	m := prior
	m.UUID = types.StringValue(t.UUID)
	m.Name = types.StringValue(t.Name)
	m.Provider = types.StringValue(t.Provider)
	m.Token = keepPriorIfHidden(t.Token, prior.Token)
	return m
}
