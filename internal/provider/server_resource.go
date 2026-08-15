package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*serverResource)(nil)
	_ resource.ResourceWithConfigure   = (*serverResource)(nil)
	_ resource.ResourceWithImportState = (*serverResource)(nil)
)

// NewServerResource is registered in provider.go.
func NewServerResource() resource.Resource {
	return &serverResource{}
}

type serverResource struct {
	client *client.Client
}

type serverResourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	IP              types.String `tfsdk:"ip"`
	Port            types.Int64  `tfsdk:"port"`
	User            types.String `tfsdk:"user"`
	PrivateKeyUUID  types.String `tfsdk:"private_key_uuid"`
	IsBuildServer   types.Bool   `tfsdk:"is_build_server"`
	InstantValidate types.Bool   `tfsdk:"instant_validate"`
	ProxyType       types.String `tfsdk:"proxy_type"`
	IsReachable     types.Bool   `tfsdk:"is_reachable"`
	IsUsable        types.Bool   `tfsdk:"is_usable"`
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Registers an existing machine as a Coolify server over SSH. " +
			"Coolify connects with the referenced private key and installs its agent; the machine " +
			"itself must already exist (this resource does not provision VMs). Deleting the " +
			"resource deregisters the server from Coolify without touching the machine.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the server.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description. Coolify only accepts letters (including " +
					"Unicode), numbers, whitespace, and `- _ . , ! ? ( ) ' \" + = * @ / &` — other " +
					"punctuation (e.g. a colon or semicolon) is rejected with a 422.",
				Optional: true,
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "IP address or hostname Coolify uses to SSH into the machine.",
				Required:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "SSH port. Defaults to `22`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(22),
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "SSH user. Defaults to `root`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("root"),
			},
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the Coolify private key used for the SSH connection.",
				Required:            true,
			},
			"is_build_server": schema.BoolAttribute{
				MarkdownDescription: "Use this server only for building images. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"instant_validate": schema.BoolAttribute{
				MarkdownDescription: "Ask Coolify to validate SSH connectivity right after creation. " +
					"Defaults to `true`. Coolify dispatches this validation as an asynchronous " +
					"background job — creation itself always succeeds immediately regardless of " +
					"this flag, and `is_reachable`/`is_usable` will read back `false` right after " +
					"`apply` even with a good key and IP; they only reflect the real state once " +
					"the background check finishes, visible on a later `plan`/`refresh`. Set to " +
					"`false` to skip queuing that background check entirely (e.g. for a server " +
					"that isn't SSH-reachable yet).",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"proxy_type": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy to run on the server: `traefik`, `caddy` or `none`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("traefik", "caddy", "none"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_reachable": schema.BoolAttribute{
				MarkdownDescription: "Whether Coolify could reach the server on its last check.",
				Computed:            true,
			},
			"is_usable": schema.BoolAttribute{
				MarkdownDescription: "Whether the server is fully validated and usable for deployments.",
				Computed:            true,
			},
		},
	}
}

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.client.CreateServer(ctx, client.ServerCreateRequest{
		Name:            stringOrNil(plan.Name),
		Description:     stringOrNil(plan.Description),
		IP:              stringOrNil(plan.IP),
		Port:            int64OrNil(plan.Port),
		User:            stringOrNil(plan.User),
		PrivateKeyUUID:  stringOrNil(plan.PrivateKeyUUID),
		IsBuildServer:   boolOrNil(plan.IsBuildServer),
		InstantValidate: boolOrNil(plan.InstantValidate),
		ProxyType:       stringOrNil(plan.ProxyType),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify server", err.Error())
		return
	}

	tflog.Trace(ctx, "created server", map[string]any{"uuid": server.UUID})

	resp.Diagnostics.Append(resp.State.Set(ctx, serverToModel(server, plan))...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify server %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, serverToModel(server, state))...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.client.UpdateServer(ctx, plan.UUID.ValueString(), client.ServerUpdateRequest{
		ServerCreateRequest: client.ServerCreateRequest{
			Name:            stringOrNil(plan.Name),
			Description:     stringOrNil(plan.Description),
			IP:              stringOrNil(plan.IP),
			Port:            int64OrNil(plan.Port),
			User:            stringOrNil(plan.User),
			PrivateKeyUUID:  stringOrNil(plan.PrivateKeyUUID),
			IsBuildServer:   boolOrNil(plan.IsBuildServer),
			InstantValidate: boolOrNil(plan.InstantValidate),
			ProxyType:       stringOrNil(plan.ProxyType),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify server %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, serverToModel(server, plan))...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify server %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState supports `terraform import coolify_server.example <uuid>`.
func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// serverToModel converts an API object to state. Write-only attributes the API
// never echoes back (private_key_uuid, instant_validate) are carried over from
// the prior state/plan to avoid phantom diffs.
func serverToModel(s *client.Server, prior serverResourceModel) serverResourceModel {
	m := serverResourceModel{
		UUID:            types.StringValue(s.UUID),
		Name:            types.StringValue(s.Name),
		IP:              types.StringValue(s.IP),
		Port:            types.Int64Value(s.Port),
		User:            types.StringValue(s.User),
		ProxyType:       types.StringValue(s.ProxyType),
		PrivateKeyUUID:  prior.PrivateKeyUUID,
		InstantValidate: prior.InstantValidate,
		IsBuildServer:   types.BoolValue(false),
		IsReachable:     types.BoolValue(false),
		IsUsable:        types.BoolValue(false),
	}
	if s.Description != "" || !prior.Description.IsNull() {
		m.Description = types.StringValue(s.Description)
	} else {
		m.Description = types.StringNull()
	}
	if s.Settings != nil {
		m.IsBuildServer = types.BoolValue(s.Settings.IsBuildServer)
		m.IsReachable = types.BoolValue(s.Settings.IsReachable)
		m.IsUsable = types.BoolValue(s.Settings.IsUsable)
	}
	return m
}
