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
	_ resource.Resource                = (*volumeBackupResource)(nil)
	_ resource.ResourceWithConfigure   = (*volumeBackupResource)(nil)
	_ resource.ResourceWithImportState = (*volumeBackupResource)(nil)
)

// NewVolumeBackupResource is registered in provider.go.
func NewVolumeBackupResource() resource.Resource {
	return &volumeBackupResource{}
}

type volumeBackupResource struct {
	client *client.Client
}

type volumeBackupResourceModel struct {
	UUID               types.String `tfsdk:"uuid"`
	ParentType         types.String `tfsdk:"parent_type"`
	ParentUUID         types.String `tfsdk:"parent_uuid"`
	StorageUUID        types.String `tfsdk:"storage_uuid"`
	Frequency          types.String `tfsdk:"frequency"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	SaveS3             types.Bool   `tfsdk:"save_s3"`
	S3StorageUUID      types.String `tfsdk:"s3_storage_uuid"`
	DisableLocalBackup types.Bool   `tfsdk:"disable_local_backup"`
	StopDuringBackup   types.Bool   `tfsdk:"stop_during_backup"`
	Timeout            types.Int64  `tfsdk:"timeout"`

	RetentionAmountLocally types.Int64 `tfsdk:"retention_amount_locally"`
	RetentionDaysLocally   types.Int64 `tfsdk:"retention_days_locally"`
	RetentionAmountS3      types.Int64 `tfsdk:"retention_amount_s3"`
	RetentionDaysS3        types.Int64 `tfsdk:"retention_days_s3"`
}

func (r *volumeBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_backup"
}

func (r *volumeBackupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A backup schedule for a persistent storage (`coolify_storage`) of an " +
			"application, service or database. One schedule per storage (server-side upsert).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier of the schedule.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"parent_type": schema.StringAttribute{
				MarkdownDescription: "`application`, `service` or `database`. Changing it forces replacement.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("application", "service", "database")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"parent_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the parent resource. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"storage_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the `coolify_storage` to back up. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "Cron expression or shorthand (`@daily`…).",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the schedule is active. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"save_s3": schema.BoolAttribute{
				MarkdownDescription: "Ship backups to S3. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"s3_storage_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the `coolify_s3_storage` target (required when `save_s3`).",
				Optional:            true,
			},
			"disable_local_backup": schema.BoolAttribute{
				MarkdownDescription: "Skip keeping a local copy. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"stop_during_backup": schema.BoolAttribute{
				MarkdownDescription: "Stop the resource while the volume is snapshotted. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Backup timeout in seconds (60–36000).",
				Optional:            true,
				Computed:            true,
			},
			"retention_amount_locally": schema.Int64Attribute{
				MarkdownDescription: "Number of local backups to keep (0 = unlimited).",
				Optional:            true,
				Computed:            true,
			},
			"retention_days_locally": schema.Int64Attribute{
				MarkdownDescription: "Days to keep local backups (0 = unlimited).",
				Optional:            true,
				Computed:            true,
			},
			"retention_amount_s3": schema.Int64Attribute{
				MarkdownDescription: "Number of S3 backups to keep (0 = unlimited).",
				Optional:            true,
				Computed:            true,
			},
			"retention_days_s3": schema.Int64Attribute{
				MarkdownDescription: "Days to keep S3 backups (0 = unlimited).",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *volumeBackupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func volumeBackupToRequest(m volumeBackupResourceModel) client.VolumeBackupScheduleRequest {
	return client.VolumeBackupScheduleRequest{
		Frequency:              stringOrNil(m.Frequency),
		Enabled:                boolOrNil(m.Enabled),
		SaveS3:                 boolOrNil(m.SaveS3),
		S3StorageUUID:          stringOrNil(m.S3StorageUUID),
		DisableLocalBackup:     boolOrNil(m.DisableLocalBackup),
		StopDuringBackup:       boolOrNil(m.StopDuringBackup),
		Timeout:                int64OrNil(m.Timeout),
		RetentionAmountLocally: int64OrNil(m.RetentionAmountLocally),
		RetentionDaysLocally:   int64OrNil(m.RetentionDaysLocally),
		RetentionAmountS3:      int64OrNil(m.RetentionAmountS3),
		RetentionDaysS3:        int64OrNil(m.RetentionDaysS3),
	}
}

func (r *volumeBackupResource) parent(m volumeBackupResourceModel) (client.EnvVarParent, string, string) {
	return client.EnvVarParent(m.ParentType.ValueString()), m.ParentUUID.ValueString(), m.StorageUUID.ValueString()
}

// upsert is shared by Create and Update: the API endpoint is a PUT.
func (r *volumeBackupResource) upsert(ctx context.Context, plan volumeBackupResourceModel) (volumeBackupResourceModel, error) {
	parent, parentUUID, storageUUID := r.parent(plan)
	schedule, err := r.client.UpsertVolumeBackupSchedule(ctx, parent, parentUUID, storageUUID, volumeBackupToRequest(plan))
	if err != nil {
		return plan, err
	}
	return volumeBackupToModel(schedule, plan), nil
}

func (r *volumeBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.upsert(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify volume backup schedule", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Read re-applies nothing: the API has no GET for volume backup schedules, so
// state is refreshed from the last upsert response. Drift on the schedule is
// reconciled on the next apply.
func (r *volumeBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *volumeBackupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan volumeBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.upsert(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coolify volume backup schedule", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *volumeBackupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID, storageUUID := r.parent(state)
	err := r.client.DeleteVolumeBackupSchedule(ctx, parent, parentUUID, storageUUID)
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coolify volume backup schedule", err.Error())
	}
}

// ImportState expects "<parent_type>/<parent_uuid>/<storage_uuid>". The
// schedule attributes are populated on the next apply (no GET endpoint).
func (r *volumeBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<parent_type>/<parent_uuid>/<storage_uuid>\", got %q.", req.ID),
		)
		return
	}
	state := volumeBackupResourceModel{
		ParentType:  types.StringValue(parts[0]),
		ParentUUID:  types.StringValue(parts[1]),
		StorageUUID: types.StringValue(parts[2]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func volumeBackupToModel(s *client.VolumeBackupSchedule, prior volumeBackupResourceModel) volumeBackupResourceModel {
	m := prior
	m.UUID = types.StringValue(s.UUID)
	m.Frequency = types.StringValue(s.Frequency)
	m.Enabled = types.BoolValue(s.Enabled)
	m.SaveS3 = types.BoolValue(s.SaveS3)
	m.DisableLocalBackup = types.BoolValue(s.DisableLocalBackup)
	m.StopDuringBackup = types.BoolValue(s.StopDuringBackup)
	m.Timeout = types.Int64Value(s.Timeout)
	m.RetentionAmountLocally = types.Int64Value(s.RetentionAmountLocally)
	m.RetentionDaysLocally = types.Int64Value(s.RetentionDaysLocally)
	m.RetentionAmountS3 = types.Int64Value(s.RetentionAmountS3)
	m.RetentionDaysS3 = types.Int64Value(s.RetentionDaysS3)
	return m
}
