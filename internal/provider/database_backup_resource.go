package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*databaseBackupResource)(nil)
	_ resource.ResourceWithConfigure   = (*databaseBackupResource)(nil)
	_ resource.ResourceWithImportState = (*databaseBackupResource)(nil)
)

// NewDatabaseBackupResource is registered in provider.go.
func NewDatabaseBackupResource() resource.Resource {
	return &databaseBackupResource{}
}

type databaseBackupResource struct {
	client *client.Client
}

type databaseBackupResourceModel struct {
	UUID              types.String `tfsdk:"uuid"`
	DatabaseUUID      types.String `tfsdk:"database_uuid"`
	Frequency         types.String `tfsdk:"frequency"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	SaveS3            types.Bool   `tfsdk:"save_s3"`
	S3StorageUUID     types.String `tfsdk:"s3_storage_uuid"`
	DumpAll           types.Bool   `tfsdk:"dump_all"`
	DatabasesToBackup types.String `tfsdk:"databases_to_backup"`
	Timeout           types.Int64  `tfsdk:"timeout"`

	RetentionAmountLocally types.Int64 `tfsdk:"retention_amount_locally"`
	RetentionDaysLocally   types.Int64 `tfsdk:"retention_days_locally"`
	RetentionAmountS3      types.Int64 `tfsdk:"retention_amount_s3"`
	RetentionDaysS3        types.Int64 `tfsdk:"retention_days_s3"`
}

func (r *databaseBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_backup"
}

func (r *databaseBackupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A scheduled backup for a standalone database (docs: databases/backups), " +
			"stored locally and optionally shipped to an S3 storage (`coolify_s3_storage`).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the database to back up. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "Cron expression (`0 3 * * *`) or shorthand (`@daily`…).",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the schedule is active. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"save_s3": schema.BoolAttribute{
				MarkdownDescription: "Also upload backups to S3. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"s3_storage_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the `coolify_s3_storage` target (required when `save_s3`).",
				Optional:            true,
			},
			"dump_all": schema.BoolAttribute{
				MarkdownDescription: "Dump the whole instance instead of selected logical databases.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"databases_to_backup": schema.StringAttribute{
				MarkdownDescription: "Comma-separated logical database names to back up.",
				Optional:            true,
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

func (r *databaseBackupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func databaseBackupToRequest(m databaseBackupResourceModel) client.DatabaseBackupRequest {
	return client.DatabaseBackupRequest{
		Frequency:              stringOrNil(m.Frequency),
		Enabled:                boolOrNil(m.Enabled),
		SaveS3:                 boolOrNil(m.SaveS3),
		S3StorageUUID:          stringOrNil(m.S3StorageUUID),
		DumpAll:                boolOrNil(m.DumpAll),
		DatabasesToBackup:      stringOrNil(m.DatabasesToBackup),
		Timeout:                int64OrNil(m.Timeout),
		RetentionAmountLocally: int64OrNil(m.RetentionAmountLocally),
		RetentionDaysLocally:   int64OrNil(m.RetentionDaysLocally),
		RetentionAmountS3:      int64OrNil(m.RetentionAmountS3),
		RetentionDaysS3:        int64OrNil(m.RetentionDaysS3),
	}
}

func (r *databaseBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backup, err := r.client.CreateDatabaseBackup(ctx, plan.DatabaseUUID.ValueString(), databaseBackupToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify database backup schedule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, databaseBackupToModel(backup, plan))...)
}

func (r *databaseBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backup, err := r.client.GetDatabaseBackup(ctx, state.DatabaseUUID.ValueString(), state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify database backup %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, databaseBackupToModel(backup, state))...)
}

func (r *databaseBackupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan databaseBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backup, err := r.client.UpdateDatabaseBackup(ctx, plan.DatabaseUUID.ValueString(), plan.UUID.ValueString(), databaseBackupToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify database backup %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, databaseBackupToModel(backup, plan))...)
}

func (r *databaseBackupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDatabaseBackup(ctx, state.DatabaseUUID.ValueString(), state.UUID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify database backup %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects "<database_uuid>/<backup_uuid>".
func (r *databaseBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<database_uuid>/<backup_uuid>\", got %q.", req.ID),
		)
		return
	}

	backup, err := r.client.GetDatabaseBackup(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify database backup", err.Error())
		return
	}

	state := databaseBackupToModel(backup, databaseBackupResourceModel{
		DatabaseUUID: types.StringValue(parts[0]),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func databaseBackupToModel(b *client.DatabaseBackup, prior databaseBackupResourceModel) databaseBackupResourceModel {
	m := prior
	m.UUID = types.StringValue(b.UUID)
	m.Frequency = types.StringValue(b.Frequency)
	m.Enabled = types.BoolValue(b.Enabled)
	m.SaveS3 = types.BoolValue(b.SaveS3)
	m.DumpAll = types.BoolValue(b.DumpAll)
	m.DatabasesToBackup = keepNullIfEmpty(b.DatabasesToBackup, prior.DatabasesToBackup)
	m.Timeout = types.Int64Value(b.Timeout)
	m.RetentionAmountLocally = types.Int64Value(b.RetentionAmountLocally)
	m.RetentionDaysLocally = types.Int64Value(b.RetentionDaysLocally)
	m.RetentionAmountS3 = types.Int64Value(b.RetentionAmountS3)
	m.RetentionDaysS3 = types.Int64Value(b.RetentionDaysS3)
	// s3_storage_uuid is not echoed (the API stores a numeric id); keep config.
	return m
}
