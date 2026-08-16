package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
				MarkdownDescription: "OAuth client id of the App. Required to create it via this " +
					"provider; not sensitive, so it's always adopted from Coolify — doesn't need to " +
					"be supplied for an imported App.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth client secret of the App. Required to create it via " +
					"this provider; a token with the `read:sensitive` ability adopts it from Coolify " +
					"on import, so it doesn't need to be supplied for an imported App.",
				Optional:      true,
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"webhook_secret": schema.StringAttribute{
				MarkdownDescription: "Webhook secret of the App. Required to create it via this " +
					"provider; a token with the `read:sensitive` ability adopts it from Coolify on " +
					"import, so it doesn't need to be supplied for an imported App.",
				Optional:      true,
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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

	model, diags := r.githubAppToModel(ctx, app, plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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

	model, diags := r.githubAppToModel(ctx, app, state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// githubAppUpdateRequest builds the PATCH body for an update, omitting
// private_key_uuid whenever it's unchanged from state.
//
// Coolify's PATCH validates private_key_uuid with Laravel's `uuid` rule (RFC
// 4122), but Coolify's own private key ids are short nanoids — no value can
// ever satisfy it (confirmed against GithubController::update(), line ~645;
// the POST path at line ~231 uses the correct `string` rule on the same
// field, so this is an upstream regression, not a real constraint). Every
// value here is unpatchable today, so only send it when it's actually
// changing — an apply that doesn't touch the private key must not 422 on a
// field nobody asked to change. A genuine private-key change is still
// attempted (and will still 422 until upstream fixes the validator) rather
// than silently dropped.
func githubAppUpdateRequest(plan, state githubAppResourceModel) client.GithubAppRequest {
	body := githubAppToRequest(plan)
	if plan.PrivateKeyUUID.ValueString() == state.PrivateKeyUUID.ValueString() {
		body.PrivateKeyUUID = nil
	}
	return body
}

func (r *githubAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state githubAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := githubAppUpdateRequest(plan, state)

	app, err := r.client.UpdateGithubApp(ctx, plan.ID.ValueInt64(), body)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify GitHub App %d", plan.ID.ValueInt64()),
			err.Error(),
		)
		return
	}

	model, diags := r.githubAppToModel(ctx, app, plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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

	model, diags := r.githubAppToModel(ctx, app, githubAppResourceModel{})
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// resolvePrivateKeyUUID looks up the uuid of the private key with the given
// numeric id. GithubApp responses only ever carry private_key_id (there's no
// single private-key GET by id, only by uuid — see PrivateKey.GetPrivateKey),
// so this always needs a full list scan, the same shape GetGithubApp itself
// needs since /github-apps has no single-object GET either.
func resolvePrivateKeyUUID(ctx context.Context, c *client.Client, id int64) (string, error) {
	keys, err := c.ListPrivateKeys(ctx)
	if err != nil {
		return "", err
	}
	for _, k := range keys {
		if k.ID == id {
			return k.UUID, nil
		}
	}
	return "", fmt.Errorf("private key id %d not found", id)
}

// githubAppToModel merges the API object with prior state. client_id,
// client_secret and webhook_secret all follow keepPriorIfHidden: Coolify
// echoes them (client_secret/webhook_secret only to a token with the
// read:sensitive ability — confirmed live: GithubApp.$hidden is overridden
// per-request by removeSensitiveData()'s makeVisible() call), so state
// adopts the real value when the token can see it and otherwise falls back
// to whatever was last configured — same treatment as every other sensitive
// field in this provider. private_key_uuid gets no such fallback: it isn't
// hidden at all, private_key_id is always present on the response and is
// resolved to its uuid here on every Create/Read/Update/Import, keeping
// state accurate instead of trusting whatever was last configured (which is
// what left it permanently null after import, triggering an update on every
// apply — see the PATCH comment in Update). Resolution failure downgrades to
// a warning and keeps the prior value, consistent with keepPriorIfHidden's
// null-preserving philosophy.
func (r *githubAppResource) githubAppToModel(ctx context.Context, a *client.GithubApp, prior githubAppResourceModel) (githubAppResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
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
	m.ClientSecret = keepPriorIfHidden(a.ClientSecret, prior.ClientSecret)
	m.WebhookSecret = keepPriorIfHidden(a.WebhookSecret, prior.WebhookSecret)
	m.IsSystemWide = types.BoolValue(a.IsSystemWide)

	if uuid, err := resolvePrivateKeyUUID(ctx, r.client, a.PrivateKeyID); err != nil {
		diags.AddWarning(
			"Unable to resolve private_key_uuid",
			fmt.Sprintf("Looking up the uuid for private key id %d failed: %s. Keeping the prior value in state.", a.PrivateKeyID, err),
		)
	} else {
		m.PrivateKeyUUID = types.StringValue(uuid)
	}

	return m, diags
}
