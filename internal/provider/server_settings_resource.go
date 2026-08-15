package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*serverSettingsResource)(nil)
	_ resource.ResourceWithConfigure   = (*serverSettingsResource)(nil)
	_ resource.ResourceWithImportState = (*serverSettingsResource)(nil)
)

// NewServerSettingsResource is registered in provider.go.
func NewServerSettingsResource() resource.Resource {
	return &serverSettingsResource{}
}

type serverSettingsResource struct {
	client *client.Client
}

type serverProxySettingsModel struct {
	RedirectEnabled     types.Bool   `tfsdk:"redirect_enabled"`
	RedirectURL         types.String `tfsdk:"redirect_url"`
	GenerateExactLabels types.Bool   `tfsdk:"generate_exact_labels"`
}

type serverDockerCleanupModel struct {
	Frequency           types.String `tfsdk:"frequency"`
	Threshold           types.Int64  `tfsdk:"threshold"`
	ForceCleanup        types.Bool   `tfsdk:"force_cleanup"`
	DeleteUnusedVolumes types.Bool   `tfsdk:"delete_unused_volumes"`
	DeleteUnusedNetwork types.Bool   `tfsdk:"delete_unused_networks"`
}

type serverSentinelModel struct {
	Enabled            types.Bool  `tfsdk:"enabled"`
	MetricsEnabled     types.Bool  `tfsdk:"metrics_enabled"`
	RefreshRateSeconds types.Int64 `tfsdk:"refresh_rate_seconds"`
	MetricsHistoryDays types.Int64 `tfsdk:"metrics_history_days"`
}

type serverCloudflareTunnelModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type serverLogDrainsModel struct {
	NewrelicEnabled    types.Bool   `tfsdk:"newrelic_enabled"`
	NewrelicLicenseKey types.String `tfsdk:"newrelic_license_key"`
	NewrelicBaseURI    types.String `tfsdk:"newrelic_base_uri"`
	AxiomEnabled       types.Bool   `tfsdk:"axiom_enabled"`
	AxiomAPIKey        types.String `tfsdk:"axiom_api_key"`
	AxiomDatasetName   types.String `tfsdk:"axiom_dataset_name"`
	CustomEnabled      types.Bool   `tfsdk:"custom_enabled"`
	CustomConfig       types.String `tfsdk:"custom_config"`
	CustomConfigParser types.String `tfsdk:"custom_config_parser"`
}

// serverSettingsResourceModel manages a server's singleton sub-settings
// (proxy, automated docker cleanup, Sentinel monitoring, Cloudflare Tunnel).
// Only the configured blocks are managed; the rest keep their values.
type serverSettingsResourceModel struct {
	ServerUUID       types.String                 `tfsdk:"server_uuid"`
	Proxy            *serverProxySettingsModel    `tfsdk:"proxy"`
	DockerCleanup    *serverDockerCleanupModel    `tfsdk:"docker_cleanup"`
	Sentinel         *serverSentinelModel         `tfsdk:"sentinel"`
	CloudflareTunnel *serverCloudflareTunnelModel `tfsdk:"cloudflare_tunnel"`
	LogDrains        *serverLogDrainsModel        `tfsdk:"log_drains"`
}

func (r *serverSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_settings"
}

func (r *serverSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Singleton settings of a Coolify server: reverse-proxy behaviour, " +
			"automated Docker cleanup (docs: knowledge-base/server/automated-cleanup), Sentinel " +
			"monitoring (docs: knowledge-base/server/sentinel) and Cloudflare Tunnel " +
			"(docs: integrations/cloudflare). Only the blocks you configure are managed; " +
			"destroying the resource leaves the server's settings as they are.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the server. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"proxy": schema.SingleNestedAttribute{
				MarkdownDescription: "Reverse-proxy behaviour.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_enabled": schema.BoolAttribute{
						MarkdownDescription: "Redirect requests for unknown hosts.",
						Optional:            true,
					},
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "URL unknown hosts are redirected to.",
						Optional:            true,
					},
					"generate_exact_labels": schema.BoolAttribute{
						MarkdownDescription: "Generate exact proxy labels.",
						Optional:            true,
					},
				},
			},
			"docker_cleanup": schema.SingleNestedAttribute{
				MarkdownDescription: "Automated Docker cleanup.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"frequency": schema.StringAttribute{
						MarkdownDescription: "Cron expression for the cleanup job.",
						Optional:            true,
					},
					"threshold": schema.Int64Attribute{
						MarkdownDescription: "Disk-usage percentage (1–99) that triggers a cleanup.",
						Optional:            true,
					},
					"force_cleanup": schema.BoolAttribute{
						MarkdownDescription: "Run cleanup on schedule regardless of the threshold.",
						Optional:            true,
					},
					"delete_unused_volumes": schema.BoolAttribute{
						MarkdownDescription: "Also delete unused volumes.",
						Optional:            true,
					},
					"delete_unused_networks": schema.BoolAttribute{
						MarkdownDescription: "Also delete unused networks.",
						Optional:            true,
					},
				},
			},
			"sentinel": schema.SingleNestedAttribute{
				MarkdownDescription: "Sentinel monitoring agent.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable Sentinel.",
						Optional:            true,
					},
					"metrics_enabled": schema.BoolAttribute{
						MarkdownDescription: "Collect metrics.",
						Optional:            true,
					},
					"refresh_rate_seconds": schema.Int64Attribute{
						MarkdownDescription: "Metrics refresh rate in seconds.",
						Optional:            true,
					},
					"metrics_history_days": schema.Int64Attribute{
						MarkdownDescription: "Days of metrics history to keep.",
						Optional:            true,
					},
				},
			},
			"cloudflare_tunnel": schema.SingleNestedAttribute{
				MarkdownDescription: "Cloudflare Tunnel (not available on the localhost server).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Mark the server as reached through a Cloudflare Tunnel.",
						Optional:            true,
					},
				},
			},
			"log_drains": schema.SingleNestedAttribute{
				MarkdownDescription: "Log drains (docs: knowledge-base/drain-logs): ship container " +
					"logs to New Relic, Axiom or a custom FluentBit output.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"newrelic_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable the New Relic drain.",
						Optional:            true,
					},
					"newrelic_license_key": schema.StringAttribute{
						MarkdownDescription: "New Relic license key.",
						Optional:            true,
						Sensitive:           true,
					},
					"newrelic_base_uri": schema.StringAttribute{
						MarkdownDescription: "New Relic log endpoint.",
						Optional:            true,
					},
					"axiom_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable the Axiom drain.",
						Optional:            true,
					},
					"axiom_api_key": schema.StringAttribute{
						MarkdownDescription: "Axiom API key.",
						Optional:            true,
						Sensitive:           true,
					},
					"axiom_dataset_name": schema.StringAttribute{
						MarkdownDescription: "Axiom dataset.",
						Optional:            true,
					},
					"custom_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable a custom FluentBit output.",
						Optional:            true,
					},
					"custom_config": schema.StringAttribute{
						MarkdownDescription: "FluentBit OUTPUT configuration.",
						Optional:            true,
					},
					"custom_config_parser": schema.StringAttribute{
						MarkdownDescription: "FluentBit PARSER configuration.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *serverSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

// apply PATCHes each configured block against its endpoint.
func (r *serverSettingsResource) apply(ctx context.Context, plan serverSettingsResourceModel) error {
	server := plan.ServerUUID.ValueString()

	setBool := func(body map[string]any, key string, v types.Bool) {
		if !v.IsNull() && !v.IsUnknown() {
			body[key] = v.ValueBool()
		}
	}
	setString := func(body map[string]any, key string, v types.String) {
		if !v.IsNull() && !v.IsUnknown() {
			body[key] = v.ValueString()
		}
	}
	setInt := func(body map[string]any, key string, v types.Int64) {
		if !v.IsNull() && !v.IsUnknown() {
			body[key] = v.ValueInt64()
		}
	}

	if p := plan.Proxy; p != nil {
		body := map[string]any{}
		setBool(body, "redirect_enabled", p.RedirectEnabled)
		setString(body, "redirect_url", p.RedirectURL)
		setBool(body, "generate_exact_labels", p.GenerateExactLabels)
		if len(body) > 0 {
			if err := r.client.UpdateServerProxy(ctx, server, body); err != nil {
				return fmt.Errorf("proxy: %w", err)
			}
		}
	}
	if d := plan.DockerCleanup; d != nil {
		body := map[string]any{}
		setString(body, "docker_cleanup_frequency", d.Frequency)
		setInt(body, "docker_cleanup_threshold", d.Threshold)
		setBool(body, "force_docker_cleanup", d.ForceCleanup)
		setBool(body, "delete_unused_volumes", d.DeleteUnusedVolumes)
		setBool(body, "delete_unused_networks", d.DeleteUnusedNetwork)
		if len(body) > 0 {
			if err := r.client.UpdateServerDockerCleanup(ctx, server, body); err != nil {
				return fmt.Errorf("docker_cleanup: %w", err)
			}
		}
	}
	if s := plan.Sentinel; s != nil {
		body := map[string]any{}
		setBool(body, "is_sentinel_enabled", s.Enabled)
		setBool(body, "is_metrics_enabled", s.MetricsEnabled)
		setInt(body, "sentinel_metrics_refresh_rate_seconds", s.RefreshRateSeconds)
		setInt(body, "sentinel_metrics_history_days", s.MetricsHistoryDays)
		if len(body) > 0 {
			if err := r.client.UpdateServerSentinel(ctx, server, body); err != nil {
				return fmt.Errorf("sentinel: %w", err)
			}
		}
	}
	if c := plan.CloudflareTunnel; c != nil {
		if !c.Enabled.IsNull() && !c.Enabled.IsUnknown() {
			body := map[string]any{"is_cloudflare_tunnel": c.Enabled.ValueBool()}
			if err := r.client.UpdateServerCloudflareTunnel(ctx, server, body); err != nil {
				return fmt.Errorf("cloudflare_tunnel: %w", err)
			}
		}
	}
	if l := plan.LogDrains; l != nil {
		body := map[string]any{}
		setBool(body, "is_logdrain_newrelic_enabled", l.NewrelicEnabled)
		setString(body, "logdrain_newrelic_license_key", l.NewrelicLicenseKey)
		setString(body, "logdrain_newrelic_base_uri", l.NewrelicBaseURI)
		setBool(body, "is_logdrain_axiom_enabled", l.AxiomEnabled)
		setString(body, "logdrain_axiom_api_key", l.AxiomAPIKey)
		setString(body, "logdrain_axiom_dataset_name", l.AxiomDatasetName)
		setBool(body, "is_logdrain_custom_enabled", l.CustomEnabled)
		setString(body, "logdrain_custom_config", l.CustomConfig)
		setString(body, "logdrain_custom_config_parser", l.CustomConfigParser)
		if len(body) > 0 {
			if err := r.client.UpdateServerLogDrains(ctx, server, body); err != nil {
				return fmt.Errorf("log_drains: %w", err)
			}
		}
	}
	return nil
}

// refresh reads back each configured block. Only fields the practitioner set
// are refreshed, so unmanaged settings never cause diffs.
func (r *serverSettingsResource) refresh(ctx context.Context, m serverSettingsResourceModel) (serverSettingsResourceModel, error) {
	server := m.ServerUUID.ValueString()

	readBool := func(settings map[string]any, key string, prior types.Bool) types.Bool {
		if prior.IsNull() {
			return prior
		}
		if v, ok := settings[key].(bool); ok {
			return types.BoolValue(v)
		}
		return prior
	}
	readString := func(settings map[string]any, key string, prior types.String) types.String {
		if prior.IsNull() {
			return prior
		}
		if v, ok := settings[key].(string); ok {
			return types.StringValue(v)
		}
		return prior
	}
	readInt := func(settings map[string]any, key string, prior types.Int64) types.Int64 {
		if prior.IsNull() {
			return prior
		}
		if v, ok := settings[key].(float64); ok {
			return types.Int64Value(int64(v))
		}
		return prior
	}

	if p := m.Proxy; p != nil {
		settings, err := r.client.GetServerProxy(ctx, server)
		if err != nil {
			return m, err
		}
		p.RedirectEnabled = readBool(settings, "redirect_enabled", p.RedirectEnabled)
		p.RedirectURL = readString(settings, "redirect_url", p.RedirectURL)
		p.GenerateExactLabels = readBool(settings, "generate_exact_labels", p.GenerateExactLabels)
	}
	if d := m.DockerCleanup; d != nil {
		settings, err := r.client.GetServerDockerCleanup(ctx, server)
		if err != nil {
			return m, err
		}
		d.Frequency = readString(settings, "docker_cleanup_frequency", d.Frequency)
		d.Threshold = readInt(settings, "docker_cleanup_threshold", d.Threshold)
		d.ForceCleanup = readBool(settings, "force_docker_cleanup", d.ForceCleanup)
		d.DeleteUnusedVolumes = readBool(settings, "delete_unused_volumes", d.DeleteUnusedVolumes)
		d.DeleteUnusedNetwork = readBool(settings, "delete_unused_networks", d.DeleteUnusedNetwork)
	}
	if s := m.Sentinel; s != nil {
		settings, err := r.client.GetServerSentinel(ctx, server)
		if err != nil {
			return m, err
		}
		s.Enabled = readBool(settings, "is_sentinel_enabled", s.Enabled)
		s.MetricsEnabled = readBool(settings, "is_metrics_enabled", s.MetricsEnabled)
		s.RefreshRateSeconds = readInt(settings, "sentinel_metrics_refresh_rate_seconds", s.RefreshRateSeconds)
		s.MetricsHistoryDays = readInt(settings, "sentinel_metrics_history_days", s.MetricsHistoryDays)
	}
	if c := m.CloudflareTunnel; c != nil {
		settings, err := r.client.GetServerCloudflareTunnel(ctx, server)
		if err != nil {
			return m, err
		}
		c.Enabled = readBool(settings, "is_cloudflare_tunnel", c.Enabled)
	}
	if l := m.LogDrains; l != nil {
		settings, err := r.client.GetServerLogDrains(ctx, server)
		if err != nil {
			return m, err
		}
		l.NewrelicEnabled = readBool(settings, "is_logdrain_newrelic_enabled", l.NewrelicEnabled)
		l.NewrelicBaseURI = readString(settings, "logdrain_newrelic_base_uri", l.NewrelicBaseURI)
		l.AxiomEnabled = readBool(settings, "is_logdrain_axiom_enabled", l.AxiomEnabled)
		l.AxiomDatasetName = readString(settings, "logdrain_axiom_dataset_name", l.AxiomDatasetName)
		l.CustomEnabled = readBool(settings, "is_logdrain_custom_enabled", l.CustomEnabled)
		l.CustomConfig = readString(settings, "logdrain_custom_config", l.CustomConfig)
		l.CustomConfigParser = readString(settings, "logdrain_custom_config_parser", l.CustomConfigParser)
		// API keys are hidden without read:sensitive; keep configured values.
	}
	return m, nil
}

func (r *serverSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Unable to apply Coolify server settings", err.Error())
		return
	}
	state, err := r.refresh(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read back Coolify server settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *serverSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.refresh(ctx, state)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify server settings for %s", state.ServerUUID.ValueString()),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *serverSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Unable to update Coolify server settings", err.Error())
		return
	}
	state, err := r.refresh(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read back Coolify server settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Delete leaves the server's settings untouched: they are singletons owned by
// the server, not by this resource.
func (r *serverSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState expects the server UUID; blocks must then be configured to be
// managed.
func (r *serverSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}
