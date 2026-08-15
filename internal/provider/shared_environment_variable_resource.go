package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*sharedEnvVarResource)(nil)
	_ resource.ResourceWithConfigure   = (*sharedEnvVarResource)(nil)
	_ resource.ResourceWithImportState = (*sharedEnvVarResource)(nil)
)

// NewSharedEnvVarResource is registered in provider.go.
func NewSharedEnvVarResource() resource.Resource {
	return &sharedEnvVarResource{}
}

type sharedEnvVarResource struct {
	client *client.Client
}

type sharedEnvVarResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Scope       types.String `tfsdk:"scope"`
	ProjectUUID types.String `tfsdk:"project_uuid"`
	Environment types.String `tfsdk:"environment"`
	ServerUUID  types.String `tfsdk:"server_uuid"`
	Key         types.String `tfsdk:"key"`
	Value       types.String `tfsdk:"value"`
	Comment     types.String `tfsdk:"comment"`
	IsLiteral   types.Bool   `tfsdk:"is_literal"`
	IsMultiline types.Bool   `tfsdk:"is_multiline"`
}

func (r *sharedEnvVarResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_environment_variable"
}

func (r *sharedEnvVarResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A shared environment variable, usable as `{{team.KEY}}`, " +
			"`{{project.KEY}}`, `{{environment.KEY}}` or `{{server.KEY}}` in resources. The `scope` " +
			"attribute selects where it lives.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Server-assigned numeric identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "`team`, `project`, `environment` or `server`. Changing it forces replacement.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("team", "project", "environment", "server")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "Project UUID (required for `project` and `environment` scopes).",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "Environment name or UUID (required for the `environment` scope).",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "Server UUID (required for the `server` scope).",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Variable name, unique within the scope. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
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
			"is_literal": schema.BoolAttribute{
				MarkdownDescription: "Treat `$` sequences literally. Defaults to `false`.",
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
		},
	}
}

func (r *sharedEnvVarResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func sharedEnvScope(m sharedEnvVarResourceModel) client.SharedEnvScope {
	return client.SharedEnvScope{
		Kind:        m.Scope.ValueString(),
		ProjectUUID: m.ProjectUUID.ValueString(),
		Environment: m.Environment.ValueString(),
		ServerUUID:  m.ServerUUID.ValueString(),
	}
}

func sharedEnvVarToRequest(m sharedEnvVarResourceModel) client.SharedEnvVarRequest {
	return client.SharedEnvVarRequest{
		Key:         stringOrNil(m.Key),
		Value:       stringOrNil(m.Value),
		Comment:     stringOrNil(m.Comment),
		IsLiteral:   boolOrNil(m.IsLiteral),
		IsMultiline: boolOrNil(m.IsMultiline),
	}
}

func (r *sharedEnvVarResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sharedEnvVarResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	v, err := r.client.CreateSharedEnvVar(ctx, sharedEnvScope(plan), sharedEnvVarToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify shared environment variable", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, sharedEnvVarToModel(v, plan))...)
}

func (r *sharedEnvVarResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sharedEnvVarResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	v, err := r.client.GetSharedEnvVar(ctx, sharedEnvScope(state), state.ID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify shared environment variable %s", state.Key.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, sharedEnvVarToModel(v, state))...)
}

func (r *sharedEnvVarResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan sharedEnvVarResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	v, err := r.client.UpdateSharedEnvVar(ctx, sharedEnvScope(plan), plan.ID.ValueInt64(), sharedEnvVarToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify shared environment variable %s", plan.Key.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, sharedEnvVarToModel(v, plan))...)
}

func (r *sharedEnvVarResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sharedEnvVarResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSharedEnvVar(ctx, sharedEnvScope(state), state.ID.ValueInt64())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify shared environment variable %s", state.Key.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects one of:
//
//	team/<key>
//	project/<project_uuid>/<key>
//	environment/<project_uuid>/<environment>/<key>
//	server/<server_uuid>/<key>
func (r *sharedEnvVarResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	prior := sharedEnvVarResourceModel{}
	var key string

	switch {
	case len(parts) == 2 && parts[0] == "team":
		prior.Scope = types.StringValue("team")
		key = parts[1]
	case len(parts) == 3 && parts[0] == "project":
		prior.Scope = types.StringValue("project")
		prior.ProjectUUID = types.StringValue(parts[1])
		key = parts[2]
	case len(parts) == 4 && parts[0] == "environment":
		prior.Scope = types.StringValue("environment")
		prior.ProjectUUID = types.StringValue(parts[1])
		prior.Environment = types.StringValue(parts[2])
		key = parts[3]
	case len(parts) == 3 && parts[0] == "server":
		prior.Scope = types.StringValue("server")
		prior.ServerUUID = types.StringValue(parts[1])
		key = parts[2]
	default:
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			"Expected team/<key>, project/<project_uuid>/<key>, "+
				"environment/<project_uuid>/<environment>/<key> or server/<server_uuid>/<key>, got "+strconv.Quote(req.ID)+".",
		)
		return
	}

	vars, err := r.client.ListSharedEnvVars(ctx, sharedEnvScope(prior))
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify shared environment variable", err.Error())
		return
	}
	for _, v := range vars {
		if v.Key == key {
			resp.Diagnostics.Append(resp.State.Set(ctx, sharedEnvVarToModel(&v, prior))...)
			return
		}
	}
	resp.Diagnostics.AddError(
		"Shared environment variable not found",
		fmt.Sprintf("No variable with key %q in scope %s.", key, prior.Scope.ValueString()),
	)
}

func sharedEnvVarToModel(v *client.SharedEnvVar, prior sharedEnvVarResourceModel) sharedEnvVarResourceModel {
	m := prior
	m.ID = types.Int64Value(v.ID)
	m.Key = types.StringValue(v.Key)
	m.IsLiteral = types.BoolValue(v.IsLiteral)
	m.IsMultiline = types.BoolValue(v.IsMultiline)
	m.Comment = keepNullIfEmpty(v.Comment, prior.Comment)
	m.Value = keepPriorIfHidden(v.Value, prior.Value)
	return m
}
