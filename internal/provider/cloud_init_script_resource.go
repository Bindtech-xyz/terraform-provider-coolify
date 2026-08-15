package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*cloudInitScriptResource)(nil)
	_ resource.ResourceWithConfigure   = (*cloudInitScriptResource)(nil)
	_ resource.ResourceWithImportState = (*cloudInitScriptResource)(nil)
)

// NewCloudInitScriptResource is registered in provider.go.
func NewCloudInitScriptResource() resource.Resource {
	return &cloudInitScriptResource{}
}

type cloudInitScriptResource struct {
	client *client.Client
}

type cloudInitScriptResourceModel struct {
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
	Script types.String `tfsdk:"script"`
}

func (r *cloudInitScriptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_init_script"
}

func (r *cloudInitScriptResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A reusable cloud-init YAML document applied to VMs provisioned by " +
			"`coolify_cloud_server`. The script must be valid cloud-init YAML " +
			"(starting with `#cloud-config`).",
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
			"script": schema.StringAttribute{
				MarkdownDescription: "cloud-init YAML content.",
				Required:            true,
			},
		},
	}
}

func (r *cloudInitScriptResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *cloudInitScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudInitScriptResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script, err := r.client.CreateCloudInitScript(ctx, client.CloudInitScriptRequest{
		Name:   stringOrNil(plan.Name),
		Script: stringOrNil(plan.Script),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify cloud-init script", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudInitScriptToModel(script, plan))...)
}

func (r *cloudInitScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudInitScriptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script, err := r.client.GetCloudInitScript(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify cloud-init script %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudInitScriptToModel(script, state))...)
}

func (r *cloudInitScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudInitScriptResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script, err := r.client.UpdateCloudInitScript(ctx, plan.UUID.ValueString(), client.CloudInitScriptRequest{
		Name:   stringOrNil(plan.Name),
		Script: stringOrNil(plan.Script),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify cloud-init script %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudInitScriptToModel(script, plan))...)
}

func (r *cloudInitScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudInitScriptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteCloudInitScript(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify cloud-init script %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

func (r *cloudInitScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// cloudInitScriptToModel echoes the configured script back into state rather
// than adopting the API's value. Coolify normalizes stored script content
// (observed: a trailing newline is stripped), and `script` is Required (not
// Computed, since Required+Computed is not a valid combination) — so any
// byte-for-byte difference between the known planned value and what Create
// returns is a hard "provider produced inconsistent result" error. Since the
// normalization is purely cosmetic, config is authoritative on every
// create/read/update; only on import (where the prior model has no script
// yet) is the API's value adopted.
func cloudInitScriptToModel(s *client.CloudInitScript, prior cloudInitScriptResourceModel) cloudInitScriptResourceModel {
	m := prior
	m.UUID = types.StringValue(s.UUID)
	m.Name = types.StringValue(s.Name)
	if prior.Script.IsNull() || prior.Script.IsUnknown() {
		m.Script = types.StringValue(s.Script)
	}
	return m
}
