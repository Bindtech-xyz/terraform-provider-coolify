package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*gitlabAppResource)(nil)
	_ resource.ResourceWithConfigure   = (*gitlabAppResource)(nil)
	_ resource.ResourceWithImportState = (*gitlabAppResource)(nil)
)

// NewGitlabAppResource is registered in provider.go.
func NewGitlabAppResource() resource.Resource {
	return &gitlabAppResource{}
}

type gitlabAppResource struct {
	client *client.Client
}

type gitlabAppResourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	HTMLURL      types.String `tfsdk:"html_url"`
	APIURL       types.String `tfsdk:"api_url"`
	GroupName    types.String `tfsdk:"group_name"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	WebhookToken types.String `tfsdk:"webhook_token"`
	RedirectURI  types.String `tfsdk:"redirect_uri"`
	IsSystemWide types.Bool   `tfsdk:"is_system_wide"`
}

func (r *gitlabAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gitlab_app"
}

func (r *gitlabAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Registers a GitLab source with Coolify to deploy private " +
			"repositories (docs: applications/ci-cd/gitlab). Works with gitlab.com and " +
			"self-hosted GitLab.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Server-assigned numeric identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the source.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the GitLab app.",
				Required:            true,
			},
			"html_url": schema.StringAttribute{
				MarkdownDescription: "GitLab instance URL. Defaults to `https://gitlab.com`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("https://gitlab.com"),
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "GitLab API URL. Defaults to `<html_url>/api/v4` server-side.",
				Optional:            true,
				Computed:            true,
			},
			"group_name": schema.StringAttribute{
				MarkdownDescription: "Comma-separated group names to filter repositories.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "GitLab OAuth application id.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "GitLab OAuth application secret.",
				Optional:            true,
				Sensitive:           true,
			},
			"webhook_token": schema.StringAttribute{
				MarkdownDescription: "Webhook token shared with GitLab.",
				Optional:            true,
				Sensitive:           true,
			},
			"redirect_uri": schema.StringAttribute{
				MarkdownDescription: "OAuth redirect URI.",
				Optional:            true,
				Computed:            true,
			},
			"is_system_wide": schema.BoolAttribute{
				MarkdownDescription: "Make the source usable by every team. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *gitlabAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func gitlabAppToRequest(m gitlabAppResourceModel) client.GitlabAppRequest {
	return client.GitlabAppRequest{
		Name:         stringOrNil(m.Name),
		HTMLURL:      stringOrNil(m.HTMLURL),
		APIURL:       stringOrNil(m.APIURL),
		GroupName:    stringOrNil(m.GroupName),
		ClientID:     stringOrNil(m.ClientID),
		ClientSecret: stringOrNil(m.ClientSecret),
		WebhookToken: stringOrNil(m.WebhookToken),
		RedirectURI:  stringOrNil(m.RedirectURI),
		IsSystemWide: boolOrNil(m.IsSystemWide),
	}
}

func (r *gitlabAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitlabAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.CreateGitlabApp(ctx, gitlabAppToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to register GitLab app with Coolify", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, gitlabAppToModel(app, plan))...)
}

func (r *gitlabAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitlabAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetGitlabApp(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify GitLab app %d", state.ID.ValueInt64()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, gitlabAppToModel(app, state))...)
}

func (r *gitlabAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gitlabAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.UpdateGitlabApp(ctx, plan.ID.ValueInt64(), gitlabAppToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify GitLab app %d", plan.ID.ValueInt64()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, gitlabAppToModel(app, plan))...)
}

func (r *gitlabAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitlabAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteGitlabApp(ctx, state.ID.ValueInt64()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify GitLab app %d", state.ID.ValueInt64()),
			err.Error(),
		)
	}
}

// ImportState expects the numeric GitLab app id.
func (r *gitlabAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected the numeric GitLab app id, got %q.", req.ID),
		)
		return
	}

	app, err := r.client.GetGitlabApp(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify GitLab app", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, gitlabAppToModel(app, gitlabAppResourceModel{}))...)
}

func gitlabAppToModel(a *client.GitlabApp, prior gitlabAppResourceModel) gitlabAppResourceModel {
	m := prior
	m.ID = types.Int64Value(a.ID)
	m.UUID = types.StringValue(a.UUID)
	m.Name = types.StringValue(a.Name)
	m.HTMLURL = types.StringValue(a.HTMLURL)
	m.APIURL = types.StringValue(a.APIURL)
	m.GroupName = keepNullIfEmpty(a.GroupName, prior.GroupName)
	m.ClientID = keepPriorIfHidden(a.ClientID, prior.ClientID)
	m.RedirectURI = types.StringValue(a.RedirectURI)
	m.IsSystemWide = types.BoolValue(a.IsSystemWide)
	return m
}
