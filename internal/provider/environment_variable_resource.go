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
	_ resource.Resource                = (*envVarResource)(nil)
	_ resource.ResourceWithConfigure   = (*envVarResource)(nil)
	_ resource.ResourceWithImportState = (*envVarResource)(nil)
)

// NewEnvVarResource is registered in provider.go.
func NewEnvVarResource() resource.Resource {
	return &envVarResource{}
}

type envVarResource struct {
	client *client.Client
}

type envVarResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	ParentType  types.String `tfsdk:"parent_type"`
	ParentUUID  types.String `tfsdk:"parent_uuid"`
	Key         types.String `tfsdk:"key"`
	Value       types.String `tfsdk:"value"`
	Comment     types.String `tfsdk:"comment"`
	IsPreview   types.Bool   `tfsdk:"is_preview"`
	IsLiteral   types.Bool   `tfsdk:"is_literal"`
	IsMultiline types.Bool   `tfsdk:"is_multiline"`
	IsRuntime   types.Bool   `tfsdk:"is_runtime"`
	IsBuildtime types.Bool   `tfsdk:"is_buildtime"`
}

func (r *envVarResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}

func (r *envVarResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An environment variable attached to an application, a service or a " +
			"database. For team/project/environment/server-scoped variables, use " +
			"`coolify_shared_environment_variable` instead.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"parent_type": schema.StringAttribute{
				MarkdownDescription: "Kind of resource the variable belongs to: `application`, " +
					"`service` or `database`. Changing it forces replacement.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("application", "service", "database")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"parent_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the parent resource. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Variable name. Changing it forces replacement (the API " +
					"identifies variables by key).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Variable value.",
				Optional:            true,
				Sensitive:           true,
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "Free-form comment (max 256 characters).",
				Optional:            true,
			},
			"is_preview": schema.BoolAttribute{
				MarkdownDescription: "Also inject into preview deployments. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_literal": schema.BoolAttribute{
				MarkdownDescription: "Treat `$` sequences literally instead of interpolating. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_multiline": schema.BoolAttribute{
				MarkdownDescription: "Value spans multiple lines. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_runtime": schema.BoolAttribute{
				MarkdownDescription: "Available at runtime. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"is_buildtime": schema.BoolAttribute{
				MarkdownDescription: "Available at build time. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (r *envVarResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func envVarToRequest(m envVarResourceModel) client.EnvVarRequest {
	return client.EnvVarRequest{
		Key:         stringOrNil(m.Key),
		Value:       stringOrNil(m.Value),
		Comment:     stringOrNil(m.Comment),
		IsPreview:   boolOrNil(m.IsPreview),
		IsLiteral:   boolOrNil(m.IsLiteral),
		IsMultiline: boolOrNil(m.IsMultiline),
		IsRuntime:   boolOrNil(m.IsRuntime),
		IsBuildtime: boolOrNil(m.IsBuildtime),
	}
}

func (r *envVarResource) parent(m envVarResourceModel) (client.EnvVarParent, string) {
	return client.EnvVarParent(m.ParentType.ValueString()), m.ParentUUID.ValueString()
}

func (r *envVarResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan envVarResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(plan)
	v, err := r.client.CreateEnvVar(ctx, parent, parentUUID, envVarToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify environment variable", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, envVarToModel(v, plan))...)
}

func (r *envVarResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state envVarResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(state)
	v, err := r.client.GetEnvVar(ctx, parent, parentUUID, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify environment variable %s", state.Key.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, envVarToModel(v, state))...)
}

func (r *envVarResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan envVarResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(plan)
	v, err := r.client.UpdateEnvVar(ctx, parent, parentUUID, plan.UUID.ValueString(), envVarToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify environment variable %s", plan.Key.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, envVarToModel(v, plan))...)
}

func (r *envVarResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state envVarResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(state)
	err := r.client.DeleteEnvVar(ctx, parent, parentUUID, state.UUID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify environment variable %s", state.Key.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects "<parent_type>/<parent_uuid>/<key>".
func (r *envVarResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<parent_type>/<parent_uuid>/<key>\", got %q.", req.ID),
		)
		return
	}

	vars, err := r.client.ListEnvVars(ctx, client.EnvVarParent(parts[0]), parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify environment variable", err.Error())
		return
	}
	for _, v := range vars {
		if v.Key == parts[2] {
			state := envVarToModel(&v, envVarResourceModel{
				ParentType: types.StringValue(parts[0]),
				ParentUUID: types.StringValue(parts[1]),
			})
			resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
			return
		}
	}
	resp.Diagnostics.AddError(
		"Environment variable not found",
		fmt.Sprintf("No variable with key %q on %s %s.", parts[2], parts[0], parts[1]),
	)
}

func envVarToModel(v *client.EnvVar, prior envVarResourceModel) envVarResourceModel {
	m := prior
	m.UUID = types.StringValue(v.UUID)
	m.Key = types.StringValue(v.Key)
	m.IsPreview = types.BoolValue(v.IsPreview)
	m.IsLiteral = types.BoolValue(v.IsLiteral)
	m.IsMultiline = types.BoolValue(v.IsMultiline)
	m.IsRuntime = types.BoolValue(v.IsRuntime)
	m.IsBuildtime = types.BoolValue(v.IsBuildtime)
	m.Comment = keepNullIfEmpty(v.Comment, prior.Comment)
	// Values are hidden from tokens without read:sensitive; configured wins.
	m.Value = keepPriorIfHidden(v.Value, prior.Value)
	return m
}
