package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*apiSettingsResource)(nil)
	_ resource.ResourceWithConfigure   = (*apiSettingsResource)(nil)
	_ resource.ResourceWithImportState = (*apiSettingsResource)(nil)
)

// NewAPISettingsResource is registered in provider.go.
func NewAPISettingsResource() resource.Resource {
	return &apiSettingsResource{}
}

type apiSettingsResource struct {
	client *client.Client
}

type apiSettingsResourceModel struct {
	APIEnabled types.Bool `tfsdk:"api_enabled"`
	MCPEnabled types.Bool `tfsdk:"mcp_enabled"`
}

func (r *apiSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_settings"
}

func (r *apiSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages instance-level toggles for Coolify's own REST API and MCP " +
			"server (`POST /enable`, `/disable`, `/mcp/enable`, `/mcp/disable`). **Requires a root " +
			"team (team 0) API token** — every other token gets a 403. Singleton, one per instance: " +
			"there is no GET for either flag, so `Read` cannot detect drift, and `Create`/`Update` " +
			"always call through to whichever endpoints match the configured values (matching " +
			"`coolify_server_settings`'s and `coolify_volume_backup`'s own documented no-read-API " +
			"constraint). **`destroy` unconditionally re-enables the API** regardless of the last " +
			"configured `api_enabled` — Coolify has no other way back in once it's off, so this " +
			"resource refuses to leave that footgun loaded — and disables the MCP server, Coolify's " +
			"own default.",
		Attributes: map[string]schema.Attribute{
			"api_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the Coolify REST API is enabled. Defaults to `true`. " +
					"Setting this to `false` locks this provider (and any other API client) out " +
					"immediately — only the Coolify UI can turn it back on from there, other than " +
					"destroying this resource.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"mcp_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the Coolify MCP server (at `/mcp`) is enabled. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *apiSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *apiSettingsResource) apply(ctx context.Context, m apiSettingsResourceModel) error {
	if m.APIEnabled.ValueBool() {
		if err := r.client.EnableAPI(ctx); err != nil {
			return fmt.Errorf("enabling API: %w", err)
		}
	} else {
		if err := r.client.DisableAPI(ctx); err != nil {
			return fmt.Errorf("disabling API: %w", err)
		}
	}
	if m.MCPEnabled.ValueBool() {
		if err := r.client.EnableMCP(ctx); err != nil {
			return fmt.Errorf("enabling MCP server: %w", err)
		}
	} else {
		if err := r.client.DisableMCP(ctx); err != nil {
			return fmt.Errorf("disabling MCP server: %w", err)
		}
	}
	return nil
}

func (r *apiSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Unable to apply Coolify instance API settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read is a no-op: neither flag has a GET, so state is authoritative and
// there is nothing to reconcile drift against — same constraint as
// coolify_volume_backup.
func (r *apiSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *apiSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Unable to apply Coolify instance API settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete re-enables the API and disables the MCP server unconditionally —
// see the schema's MarkdownDescription for why this ignores the resource's
// last configured state rather than mirroring it.
func (r *apiSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.client.EnableAPI(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to re-enable the Coolify API on destroy", err.Error())
	}
	if err := r.client.DisableMCP(ctx); err != nil {
		resp.Diagnostics.AddError("Unable to disable the Coolify MCP server on destroy", err.Error())
	}
}

// ImportState accepts any non-empty id — there is nothing in Coolify to
// look up (no GET, no identifying attribute; this is a single global
// instance setting). State is seeded with the schema defaults; the next
// plan will show a diff (and Update will run) if the real instance state
// differs from what you configure, since there is no way to read it first.
func (r *apiSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Unexpected import identifier", "Provide any non-empty id, e.g. `instance`.")
		return
	}
	state := apiSettingsResourceModel{
		APIEnabled: types.BoolValue(true),
		MCPEnabled: types.BoolValue(false),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
