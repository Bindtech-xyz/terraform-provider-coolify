package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource              = (*resourceActionResource)(nil)
	_ resource.ResourceWithConfigure = (*resourceActionResource)(nil)
)

// NewResourceActionResource is registered in provider.go.
func NewResourceActionResource() resource.Resource {
	return &resourceActionResource{}
}

type resourceActionResource struct {
	client *client.Client
}

type resourceActionResourceModel struct {
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceUUID types.String `tfsdk:"resource_uuid"`
	Action       types.String `tfsdk:"action"`
	Triggers     types.Map    `tfsdk:"triggers"`
}

func (r *resourceActionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_action"
}

func (r *resourceActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers a `start`, `stop` or `restart` on an application, database " +
			"or service. The action runs on create and again whenever `triggers` changes " +
			"(replacement re-runs it). Destroying the resource does nothing.",
		Attributes: map[string]schema.Attribute{
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "`application`, `database` or `service`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("application", "database", "service")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"resource_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the target resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "`start`, `stop` or `restart`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("start", "stop", "restart")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "Arbitrary values; any change re-runs the action " +
					"(e.g. `{ config_hash = sha1(...) }`).",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapRequiresReplace(),
				},
			},
		},
	}
}

func (r *resourceActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *resourceActionResource) run(ctx context.Context, m resourceActionResourceModel) error {
	uuid := m.ResourceUUID.ValueString()
	switch m.ResourceType.ValueString() + "/" + m.Action.ValueString() {
	case "application/start":
		return r.client.StartApplication(ctx, uuid)
	case "application/stop":
		return r.client.StopApplication(ctx, uuid)
	case "application/restart":
		return r.client.RestartApplication(ctx, uuid)
	case "database/start":
		return r.client.StartDatabase(ctx, uuid)
	case "database/stop":
		return r.client.StopDatabase(ctx, uuid)
	case "database/restart":
		return r.client.RestartDatabase(ctx, uuid)
	case "service/start":
		return r.client.StartService(ctx, uuid)
	case "service/stop":
		return r.client.StopService(ctx, uuid)
	case "service/restart":
		return r.client.RestartService(ctx, uuid)
	default:
		return fmt.Errorf("unsupported action %s on %s", m.Action.ValueString(), m.ResourceType.ValueString())
	}
}

func (r *resourceActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.run(ctx, plan); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to %s %s %s", plan.Action.ValueString(), plan.ResourceType.ValueString(), plan.ResourceUUID.ValueString()),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read is a no-op: the action is fire-and-forget.
func (r *resourceActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update never runs: every attribute forces replacement.
func (r *resourceActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete does not touch the target resource.
func (r *resourceActionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
