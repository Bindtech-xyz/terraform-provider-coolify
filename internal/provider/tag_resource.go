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
	_ resource.Resource                = (*tagResource)(nil)
	_ resource.ResourceWithConfigure   = (*tagResource)(nil)
	_ resource.ResourceWithImportState = (*tagResource)(nil)
)

// NewTagResource is registered in provider.go.
func NewTagResource() resource.Resource {
	return &tagResource{}
}

type tagResource struct {
	client *client.Client
}

type tagResourceModel struct {
	UUID types.String `tfsdk:"uuid"`
	Name types.String `tfsdk:"name"`
}

func (r *tagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *tagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A team-wide tag. Tags can be attached to applications, services and " +
			"databases, and drive batch deployments via the `/deploy?tag=` API.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Tag name (2 to 255 characters).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(2, 255)},
			},
		},
	}
}

func (r *tagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *tagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.client.CreateTag(ctx, client.TagRequest{Name: stringOrNil(plan.Name)})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify tag", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tagResourceModel{
		UUID: types.StringValue(tag.UUID),
		Name: types.StringValue(tag.Name),
	})...)
}

func (r *tagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.client.GetTag(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify tag %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tagResourceModel{
		UUID: types.StringValue(tag.UUID),
		Name: types.StringValue(tag.Name),
	})...)
}

func (r *tagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.client.UpdateTag(ctx, plan.UUID.ValueString(), client.TagRequest{Name: stringOrNil(plan.Name)})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify tag %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tagResourceModel{
		UUID: types.StringValue(tag.UUID),
		Name: types.StringValue(tag.Name),
	})...)
}

func (r *tagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTag(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify tag %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

func (r *tagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
