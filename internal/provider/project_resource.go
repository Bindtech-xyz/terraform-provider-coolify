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
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*projectResource)(nil)
	_ resource.ResourceWithConfigure   = (*projectResource)(nil)
	_ resource.ResourceWithImportState = (*projectResource)(nil)
)

// NewProjectResource is registered in provider.go.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *client.Client
}

// projectResourceModel maps the coolify_project schema. Field order follows the
// schema for readability.
type projectResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Coolify project. Projects group environments, which in turn " +
			"hold applications, databases and services.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier of the project.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					// The UUID never changes for the life of the project; keep
					// plans free of "(known after apply)" noise.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description. Coolify only accepts letters (including " +
					"Unicode), numbers, whitespace, and `- _ . , ! ? ( ) ' \" + = * @ / &` — other " +
					"punctuation (e.g. a colon or semicolon) is rejected with a 422.",
				Optional: true,
			},
		},
	}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.CreateProject(ctx, client.ProjectRequest{
		Name:        stringOrNil(plan.Name),
		Description: stringOrNil(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify project", err.Error())
		return
	}

	tflog.Trace(ctx, "created project", map[string]any{"uuid": project.UUID})

	resp.Diagnostics.Append(resp.State.Set(ctx, projectToModel(project, plan))...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.GetProject(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Deleted outside Terraform: drop from state so the next plan
			// proposes recreation instead of erroring.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify project %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, projectToModel(project, state))...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.UpdateProject(ctx, plan.UUID.ValueString(), client.ProjectRequest{
		Name:        stringOrNil(plan.Name),
		Description: stringOrNil(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify project %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, projectToModel(project, plan))...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteProject(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify project %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState supports `terraform import coolify_project.example <uuid>`.
func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// projectToModel converts an API object to state, preserving the config's null
// vs empty-string choice for description: Coolify normalises absent descriptions
// to "", which would otherwise produce a permanent diff for practitioners who
// never set the attribute.
func projectToModel(p *client.Project, prior projectResourceModel) projectResourceModel {
	m := projectResourceModel{
		UUID: types.StringValue(p.UUID),
		Name: types.StringValue(p.Name),
	}
	if p.Description != "" || !prior.Description.IsNull() {
		m.Description = types.StringValue(p.Description)
	} else {
		m.Description = types.StringNull()
	}
	return m
}
