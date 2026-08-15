package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*storageResource)(nil)
	_ resource.ResourceWithConfigure   = (*storageResource)(nil)
	_ resource.ResourceWithImportState = (*storageResource)(nil)
)

// NewStorageResource is registered in provider.go.
func NewStorageResource() resource.Resource {
	return &storageResource{}
}

type storageResource struct {
	client *client.Client
}

type storageResourceModel struct {
	UUID       types.String `tfsdk:"uuid"`
	ParentType types.String `tfsdk:"parent_type"`
	ParentUUID types.String `tfsdk:"parent_uuid"`
	Type       types.String `tfsdk:"type"`
	Name       types.String `tfsdk:"name"`
	MountPath  types.String `tfsdk:"mount_path"`
	HostPath   types.String `tfsdk:"host_path"`
	Content    types.String `tfsdk:"content"`
	VolumeName types.String `tfsdk:"volume_name"`
}

func (r *storageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage"
}

func (r *storageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Persistent storage attached to an application, service or database " +
			"(docs: knowledge-base/persistent-storage). `type = \"persistent\"` mounts a named " +
			"Docker volume (or a host directory via `host_path`); `type = \"file\"` mounts a file " +
			"whose content is managed inline.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
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
			"type": schema.StringAttribute{
				MarkdownDescription: "`persistent` (volume) or `file` (file mount). Changing it forces replacement.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("persistent", "file")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Volume name (required for `persistent`, invalid for `file`). " +
					"Coolify prefixes it with the parent's UUID server-side (`<parent-uuid>-<name>`) " +
					"— that real, effective name is exposed separately as `volume_name`; this " +
					"attribute always reflects what you configured, unchanged.",
				Optional: true,
			},
			"mount_path": schema.StringAttribute{
				MarkdownDescription: "Path inside the container.",
				Required:            true,
			},
			"host_path": schema.StringAttribute{
				MarkdownDescription: "Host directory to bind-mount instead of a named volume (`persistent` only).",
				Optional:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "File content (`file` only). Write-only: Coolify's API never " +
					"echoes this back (the underlying field is unconditionally hidden, independent " +
					"of the token's `read:sensitive` ability), so this always reflects the last " +
					"value you configured, not the live content on the server.",
				Optional: true,
			},
			"volume_name": schema.StringAttribute{
				MarkdownDescription: "The real Docker volume name Coolify assigns — `name` prefixed " +
					"with the parent resource's UUID. Empty for `file` mounts, which have no " +
					"underlying named volume.",
				Computed: true,
			},
		},
	}
}

func (r *storageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func storageToRequest(m storageResourceModel) client.StorageRequest {
	return client.StorageRequest{
		Type:      stringOrNil(m.Type),
		Name:      stringOrNil(m.Name),
		MountPath: stringOrNil(m.MountPath),
		HostPath:  stringOrNil(m.HostPath),
		Content:   stringOrNil(m.Content),
	}
}

func (r *storageResource) parent(m storageResourceModel) (client.EnvVarParent, string) {
	return client.EnvVarParent(m.ParentType.ValueString()), m.ParentUUID.ValueString()
}

func (r *storageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(plan)
	s, err := r.client.CreateStorage(ctx, parent, parentUUID, storageToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify storage", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, storageToModel(s, plan))...)
}

func (r *storageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(state)
	s, err := r.client.GetStorage(ctx, parent, parentUUID, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify storage %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, storageToModel(s, state))...)
}

func (r *storageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan storageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(plan)
	s, err := r.client.UpdateStorage(ctx, parent, parentUUID, plan.UUID.ValueString(), storageToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify storage %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, storageToModel(s, plan))...)
}

func (r *storageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(state)
	err := r.client.DeleteStorage(ctx, parent, parentUUID, state.UUID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify storage %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects "<parent_type>/<parent_uuid>/<storage_uuid>".
func (r *storageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<parent_type>/<parent_uuid>/<storage_uuid>\", got %q.", req.ID),
		)
		return
	}

	s, err := r.client.GetStorage(ctx, client.EnvVarParent(parts[0]), parts[1], parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify storage", err.Error())
		return
	}

	state := storageToModel(s, storageResourceModel{
		ParentType: types.StringValue(parts[0]),
		ParentUUID: types.StringValue(parts[1]),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// storageToModel adopts ONLY uuid and volume_name from the API. Everything
// else about this resource turns out to be unreliable to read back:
//
//   - type has no corresponding response field at all — Coolify models
//     "persistent" and "file" storages as two entirely separate Eloquent
//     models (LocalPersistentVolume, LocalFileVolume) with no shared
//     discriminator column, not as one row with a type field. It is
//     RequiresReplace anyway, so echoing the configured value is exact by
//     construction.
//   - name is echoed unchanged — the real, effective name Coolify assigns
//     server-side (uuid-prefixed) is a DIFFERENT string, surfaced instead as
//     volume_name.
//   - content is unconditionally hidden server-side (LocalFileVolume's
//     $hidden, independent of read:sensitive) — the API never returns it
//     under any circumstance, so there is nothing to adopt.
//   - mount_path and host_path are normalised server-side (trimmed, forced
//     to start with "/") — adopting them risks the exact same
//     planned-vs-final divergence already hit on name, for a cosmetic gain
//     not worth that risk.
//
// Adopting any of these from the API produced "provider produced
// inconsistent result" errors during a real deployment (name) or silently
// wrong values (type always "", content always "").
func storageToModel(s *client.Storage, prior storageResourceModel) storageResourceModel {
	m := prior
	m.UUID = types.StringValue(s.UUID)
	if s.Name != "" {
		m.VolumeName = types.StringValue(s.Name)
	} else {
		m.VolumeName = types.StringNull()
	}
	return m
}
