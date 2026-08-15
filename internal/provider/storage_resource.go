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

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
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
				MarkdownDescription: "Volume name (required for `persistent`, invalid for `file`).",
				Optional:            true,
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
				MarkdownDescription: "File content (`file` only).",
				Optional:            true,
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

func storageToModel(s *client.Storage, prior storageResourceModel) storageResourceModel {
	m := prior
	m.UUID = types.StringValue(s.UUID)
	m.Type = types.StringValue(s.Type)
	m.MountPath = types.StringValue(s.MountPath)
	m.Name = keepNullIfEmpty(s.Name, prior.Name)
	m.HostPath = keepNullIfEmpty(s.HostPath, prior.HostPath)
	m.Content = keepPriorIfHidden(s.Content, prior.Content)
	if m.Content.IsUnknown() {
		m.Content = types.StringNull()
	}
	return m
}
