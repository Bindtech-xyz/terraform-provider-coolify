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

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*environmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*environmentResource)(nil)
	_ resource.ResourceWithImportState = (*environmentResource)(nil)
)

// NewEnvironmentResource is registered in provider.go.
func NewEnvironmentResource() resource.Resource {
	return &environmentResource{}
}

type environmentResource struct {
	client *client.Client
}

type environmentResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	ProjectUUID types.String `tfsdk:"project_uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *environmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *environmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An environment inside a Coolify project (e.g. `production`, `staging`). " +
			"Applications, databases and services are always placed in an environment. " +
			"Note that Coolify creates a default `production` environment with every project.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the parent project. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the environment, unique within the project.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description.",
				Optional:            true,
			},
		},
	}
}

func (r *environmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env, err := r.client.CreateEnvironment(ctx, plan.ProjectUUID.ValueString(), client.EnvironmentRequest{
		Name:        stringOrNil(plan.Name),
		Description: stringOrNil(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify environment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, environmentToModel(env, plan))...)
}

func (r *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env, err := r.client.GetEnvironment(ctx, state.ProjectUUID.ValueString(), state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify environment %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, environmentToModel(env, state))...)
}

func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env, err := r.client.UpdateEnvironment(ctx, plan.ProjectUUID.ValueString(), plan.UUID.ValueString(), client.EnvironmentRequest{
		Name:        stringOrNil(plan.Name),
		Description: stringOrNil(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify environment %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, environmentToModel(env, plan))...)
}

func (r *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEnvironment(ctx, state.ProjectUUID.ValueString(), state.UUID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify environment %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects "<project_uuid>/<environment_name_or_uuid>".
func (r *environmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<project_uuid>/<environment_name_or_uuid>\", got %q.", req.ID),
		)
		return
	}

	env, err := r.client.GetEnvironment(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify environment", err.Error())
		return
	}

	state := environmentToModel(env, environmentResourceModel{
		ProjectUUID: types.StringValue(parts[0]),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func environmentToModel(e *client.Environment, prior environmentResourceModel) environmentResourceModel {
	m := environmentResourceModel{
		UUID:        types.StringValue(e.UUID),
		ProjectUUID: prior.ProjectUUID,
		Name:        types.StringValue(e.Name),
	}
	if e.Description != "" || !prior.Description.IsNull() {
		m.Description = types.StringValue(e.Description)
	} else {
		m.Description = types.StringNull()
	}
	return m
}
