package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                     = (*serviceResource)(nil)
	_ resource.ResourceWithConfigure        = (*serviceResource)(nil)
	_ resource.ResourceWithImportState      = (*serviceResource)(nil)
	_ resource.ResourceWithConfigValidators = (*serviceResource)(nil)
)

// NewServiceResource is registered in provider.go.
func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

type serviceResource struct {
	client *client.Client
}

type serviceResourceModel struct {
	UUID types.String `tfsdk:"uuid"`

	// Source: a one-click template type or a raw compose file.
	Type             types.String `tfsdk:"type"`
	DockerComposeRaw types.String `tfsdk:"docker_compose_raw"`

	// Placement.
	ProjectUUID     types.String `tfsdk:"project_uuid"`
	EnvironmentName types.String `tfsdk:"environment_name"`
	EnvironmentUUID types.String `tfsdk:"environment_uuid"`
	ServerUUID      types.String `tfsdk:"server_uuid"`
	DestinationUUID types.String `tfsdk:"destination_uuid"`

	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	InstantDeploy types.Bool   `tfsdk:"instant_deploy"`

	// Read-only.
	Status types.String `tfsdk:"status"`
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Coolify service: either a one-click service from the catalog " +
			"(set `type`, e.g. `plausible` — discover valid types dynamically with the " +
			"`coolify_service_templates` data source) or a custom docker-compose stack " +
			"(set `docker_compose_raw`).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "One-click service type (e.g. `plausible`, `gitea`, `umami`). " +
					"The catalog evolves with Coolify itself; the provider deliberately does not " +
					"hardcode the list. Changing it forces replacement.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"docker_compose_raw": schema.StringAttribute{
				MarkdownDescription: "Base64-encoded docker-compose file for a custom service " +
					"(use `base64encode(file(...))`).",
				Optional: true,
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the project. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "Environment name. Exactly one of `environment_name`/`environment_uuid`.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_uuid": schema.StringAttribute{
				MarkdownDescription: "Environment UUID.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the server. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"destination_uuid": schema.StringAttribute{
				MarkdownDescription: "Destination UUID (required when the server has several).",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Service name. Coolify generates one when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description.",
				Optional:            true,
			},
			"instant_deploy": schema.BoolAttribute{
				MarkdownDescription: "Start the service right after creation. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Runtime status reported by Coolify.",
				Computed:            true,
			},
		},
	}
}

func (r *serviceResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("type"),
			path.MatchRoot("docker_compose_raw"),
		),
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("environment_name"),
			path.MatchRoot("environment_uuid"),
		),
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.client.CreateService(ctx, client.ServiceCreateRequest{
		Type:             stringOrNil(plan.Type),
		DockerComposeRaw: stringOrNil(plan.DockerComposeRaw),
		Name:             stringOrNil(plan.Name),
		Description:      stringOrNil(plan.Description),
		ProjectUUID:      stringOrNil(plan.ProjectUUID),
		EnvironmentName:  stringOrNil(plan.EnvironmentName),
		EnvironmentUUID:  stringOrNil(plan.EnvironmentUUID),
		ServerUUID:       stringOrNil(plan.ServerUUID),
		DestinationUUID:  stringOrNil(plan.DestinationUUID),
		InstantDeploy:    boolOrNil(plan.InstantDeploy),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify service", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, serviceToModel(svc, plan))...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.client.GetService(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify service %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, serviceToModel(svc, state))...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.client.UpdateService(ctx, plan.UUID.ValueString(), client.ServiceUpdateRequest{
		Name:             stringOrNil(plan.Name),
		Description:      stringOrNil(plan.Description),
		DockerComposeRaw: stringOrNil(plan.DockerComposeRaw),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify service %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, serviceToModel(svc, plan))...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteService(ctx, state.UUID.ValueString(), nil, nil, nil, nil)
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify service %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func serviceToModel(s *client.Service, prior serviceResourceModel) serviceResourceModel {
	m := prior
	m.UUID = types.StringValue(s.UUID)
	m.Name = types.StringValue(s.Name)
	m.Status = types.StringValue(s.Status)
	m.Description = keepNullIfEmpty(s.Description, prior.Description)
	// service_type is echoed for one-click services; keep null for compose ones.
	if !prior.Type.IsNull() || s.Type != "" {
		m.Type = keepNullIfEmpty(s.Type, prior.Type)
	}
	return m
}
