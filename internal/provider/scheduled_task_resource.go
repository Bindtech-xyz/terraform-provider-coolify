package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*scheduledTaskResource)(nil)
	_ resource.ResourceWithConfigure   = (*scheduledTaskResource)(nil)
	_ resource.ResourceWithImportState = (*scheduledTaskResource)(nil)
)

// NewScheduledTaskResource is registered in provider.go.
func NewScheduledTaskResource() resource.Resource {
	return &scheduledTaskResource{}
}

type scheduledTaskResource struct {
	client *client.Client
}

type scheduledTaskResourceModel struct {
	UUID       types.String `tfsdk:"uuid"`
	ParentType types.String `tfsdk:"parent_type"`
	ParentUUID types.String `tfsdk:"parent_uuid"`
	Name       types.String `tfsdk:"name"`
	Command    types.String `tfsdk:"command"`
	Frequency  types.String `tfsdk:"frequency"`
	Container  types.String `tfsdk:"container"`
	Timeout    types.Int64  `tfsdk:"timeout"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func (r *scheduledTaskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_task"
}

func (r *scheduledTaskResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A scheduled task (cron job) executed inside an application's or " +
			"service's container (docs: knowledge-base/cron-syntax).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"parent_type": schema.StringAttribute{
				MarkdownDescription: "`application` or `service`. Changing it forces replacement.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("application", "service")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"parent_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the parent resource. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Task name.",
				Required:            true,
			},
			"command": schema.StringAttribute{
				MarkdownDescription: "Command to run.",
				Required:            true,
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "Cron expression (`0 3 * * *`) or shorthand (`@daily`, `@hourly`…).",
				Required:            true,
			},
			"container": schema.StringAttribute{
				MarkdownDescription: "Container to run in (for multi-container compose resources).",
				Optional:            true,
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds.",
				Optional:            true,
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the task is active. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (r *scheduledTaskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func scheduledTaskToRequest(m scheduledTaskResourceModel) client.ScheduledTaskRequest {
	return client.ScheduledTaskRequest{
		Name:      stringOrNil(m.Name),
		Command:   stringOrNil(m.Command),
		Frequency: stringOrNil(m.Frequency),
		Container: stringOrNil(m.Container),
		Timeout:   int64OrNil(m.Timeout),
		Enabled:   boolOrNil(m.Enabled),
	}
}

func (r *scheduledTaskResource) parent(m scheduledTaskResourceModel) (client.ScheduledTaskParent, string) {
	return client.ScheduledTaskParent(m.ParentType.ValueString()), m.ParentUUID.ValueString()
}

func (r *scheduledTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduledTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(plan)
	task, err := r.client.CreateScheduledTask(ctx, parent, parentUUID, scheduledTaskToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify scheduled task", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scheduledTaskToModel(task, plan))...)
}

func (r *scheduledTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduledTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(state)
	task, err := r.client.GetScheduledTask(ctx, parent, parentUUID, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify scheduled task %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scheduledTaskToModel(task, state))...)
}

func (r *scheduledTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduledTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(plan)
	task, err := r.client.UpdateScheduledTask(ctx, parent, parentUUID, plan.UUID.ValueString(), scheduledTaskToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify scheduled task %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scheduledTaskToModel(task, plan))...)
}

func (r *scheduledTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduledTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(state)
	err := r.client.DeleteScheduledTask(ctx, parent, parentUUID, state.UUID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify scheduled task %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects "<parent_type>/<parent_uuid>/<task_uuid>".
func (r *scheduledTaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<parent_type>/<parent_uuid>/<task_uuid>\", got %q.", req.ID),
		)
		return
	}

	task, err := r.client.GetScheduledTask(ctx, client.ScheduledTaskParent(parts[0]), parts[1], parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify scheduled task", err.Error())
		return
	}

	state := scheduledTaskToModel(task, scheduledTaskResourceModel{
		ParentType: types.StringValue(parts[0]),
		ParentUUID: types.StringValue(parts[1]),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func scheduledTaskToModel(t *client.ScheduledTask, prior scheduledTaskResourceModel) scheduledTaskResourceModel {
	m := prior
	m.UUID = types.StringValue(t.UUID)
	m.Name = types.StringValue(t.Name)
	m.Command = types.StringValue(t.Command)
	m.Frequency = types.StringValue(t.Frequency)
	m.Container = keepNullIfEmpty(t.Container, prior.Container)
	m.Timeout = types.Int64Value(t.Timeout)
	m.Enabled = types.BoolValue(t.Enabled)
	return m
}
