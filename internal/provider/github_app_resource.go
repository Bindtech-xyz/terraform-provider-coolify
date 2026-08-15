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

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*githubAppResource)(nil)
	_ resource.ResourceWithConfigure   = (*githubAppResource)(nil)
	_ resource.ResourceWithImportState = (*githubAppResource)(nil)
)

// NewGithubAppResource is registered in provider.go.
func NewGithubAppResource() resource.Resource {
	return &githubAppResource{}
}

type githubAppResource struct {
	client *client.Client
}

type githubAppResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	UUID           types.String `tfsdk:"uuid"`
	Name           types.String `tfsdk:"name"`
	Organization   types.String `tfsdk:"organization"`
	APIURL         types.String `tfsdk:"api_url"`
	HTMLURL        types.String `tfsdk:"html_url"`
	AppID          types.Int64  `tfsdk:"app_id"`
	InstallationID types.Int64  `tfsdk:"installation_id"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	WebhookSecret  types.String `tfsdk:"webhook_secret"`
	PrivateKeyUUID types.String `tfsdk:"private_key_uuid"`
	IsSystemWide   types.Bool   `tfsdk:"is_system_wide"`
}

func (r *githubAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app"
}

func (r *githubAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Registers an existing GitHub App with Coolify to deploy private " +
			"repositories (docs: applications/ci-cd/github). Create the App on GitHub first " +
			"(with its private key stored as a `coolify_private_key`), then reference it from " +
			"`coolify_application.github_app_uuid`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Server-assigned numeric identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "UUID, referenced by `coolify_application.github_app_uuid`.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the GitHub App.",
				Required:            true,
			},
			"organization": schema.StringAttribute{
				MarkdownDescription: "GitHub organization the App belongs to (omit for a user App).",
				Optional:            true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "GitHub API URL. Defaults to `https://api.github.com`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("https://api.github.com"),
			},
			"html_url": schema.StringAttribute{
				MarkdownDescription: "GitHub web URL. Defaults to `https://github.com`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("https://github.com"),
			},
			"app_id": schema.Int64Attribute{
				MarkdownDescription: "Numeric App ID from GitHub.",
				Required:            true,
			},
			"installation_id": schema.Int64Attribute{
				MarkdownDescription: "Numeric installation ID from GitHub.",
				Required:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth client id of the App.",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth client secret of the App.",
				Required:            true,
				Sensitive:           true,
			},
			"webhook_secret": schema.StringAttribute{
				MarkdownDescription: "Webhook secret of the App.",
				Required:            true,
				Sensitive:           true,
			},
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the `coolify_private_key` holding the App's private key.",
				Required:            true,
			},
			"is_system_wide": schema.BoolAttribute{
				MarkdownDescription: "Make the App usable by every team. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *githubAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func githubAppToRequest(m githubAppResourceModel) client.GithubAppRequest {
	return client.GithubAppRequest{
		Name:           stringOrNil(m.Name),
		Organization:   stringOrNil(m.Organization),
		APIURL:         stringOrNil(m.APIURL),
		HTMLURL:        stringOrNil(m.HTMLURL),
		AppID:          int64OrNil(m.AppID),
		InstallationID: int64OrNil(m.InstallationID),
		ClientID:       stringOrNil(m.ClientID),
		ClientSecret:   stringOrNil(m.ClientSecret),
		WebhookSecret:  stringOrNil(m.WebhookSecret),
		PrivateKeyUUID: stringOrNil(m.PrivateKeyUUID),
		IsSystemWide:   boolOrNil(m.IsSystemWide),
	}
}

func (r *githubAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.CreateGithubApp(ctx, githubAppToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to register GitHub App with Coolify", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, githubAppToModel(app, plan))...)
}

func (r *githubAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetGithubApp(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify GitHub App %d", state.ID.ValueInt64()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, githubAppToModel(app, state))...)
}

func (r *githubAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.UpdateGithubApp(ctx, plan.ID.ValueInt64(), githubAppToRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify GitHub App %d", plan.ID.ValueInt64()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, githubAppToModel(app, plan))...)
}

func (r *githubAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteGithubApp(ctx, state.ID.ValueInt64()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify GitHub App %d", state.ID.ValueInt64()),
			err.Error(),
		)
	}
}

// ImportState expects the numeric GitHub App id (as shown in Coolify).
func (r *githubAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected the numeric GitHub App id, got %q.", req.ID),
		)
		return
	}

	app, err := r.client.GetGithubApp(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify GitHub App", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, githubAppToModel(app, githubAppResourceModel{}))...)
}

// githubAppToModel merges the API object with prior state; secrets are never
// echoed back and are preserved from configuration.
func githubAppToModel(a *client.GithubApp, prior githubAppResourceModel) githubAppResourceModel {
	m := prior
	m.ID = types.Int64Value(a.ID)
	m.UUID = types.StringValue(a.UUID)
	m.Name = types.StringValue(a.Name)
	m.Organization = keepNullIfEmpty(a.Organization, prior.Organization)
	m.APIURL = types.StringValue(a.APIURL)
	m.HTMLURL = types.StringValue(a.HTMLURL)
	m.AppID = types.Int64Value(a.AppID)
	m.InstallationID = types.Int64Value(a.InstallationID)
	m.ClientID = keepPriorIfHidden(a.ClientID, prior.ClientID)
	m.IsSystemWide = types.BoolValue(a.IsSystemWide)
	return m
}
