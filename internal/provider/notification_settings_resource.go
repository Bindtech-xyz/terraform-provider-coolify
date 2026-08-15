package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource              = (*notificationSettingsResource)(nil)
	_ resource.ResourceWithConfigure = (*notificationSettingsResource)(nil)
)

// NewNotificationSettingsResource is registered in provider.go.
func NewNotificationSettingsResource() resource.Resource {
	return &notificationSettingsResource{}
}

type notificationSettingsResource struct {
	client *client.Client
}

// notificationSettingsResourceModel manages one channel of the team's
// notification settings (docs: knowledge-base/notifications). The settings are
// a singleton per channel: Create adopts them, Delete disables the channel.
type notificationSettingsResourceModel struct {
	Channel types.String `tfsdk:"channel"`
	Enabled types.Bool   `tfsdk:"enabled"`

	// Channel credentials (set the ones matching the channel).
	WebhookURL      types.String `tfsdk:"webhook_url"`
	TelegramToken   types.String `tfsdk:"telegram_token"`
	TelegramChatID  types.String `tfsdk:"telegram_chat_id"`
	PushoverUserKey types.String `tfsdk:"pushover_user_key"`
	PushoverToken   types.String `tfsdk:"pushover_api_token"`

	// Email (smtp) credentials.
	SMTPHost        types.String `tfsdk:"smtp_host"`
	SMTPPort        types.Int64  `tfsdk:"smtp_port"`
	SMTPUsername    types.String `tfsdk:"smtp_username"`
	SMTPPassword    types.String `tfsdk:"smtp_password"`
	SMTPFromAddress types.String `tfsdk:"smtp_from_address"`
	SMTPFromName    types.String `tfsdk:"smtp_from_name"`
	SMTPRecipients  types.String `tfsdk:"smtp_recipients"`

	// Event toggles, e.g. { deployment_failure = true, backup_failure = true }.
	Events types.Map `tfsdk:"events"`
}

// notificationEnabledField maps a channel to its "enabled" API field.
func notificationEnabledField(channel string) string {
	if channel == "email" {
		return "smtp_enabled"
	}
	return channel + "_enabled"
}

// notificationWebhookField maps a channel to its webhook-url API field.
func notificationWebhookField(channel string) string {
	if channel == "webhook" {
		return "webhook_url"
	}
	return channel + "_webhook_url"
}

// notificationEventField renders an event key ("deployment_failure") into the
// channel's API field ("deployment_failure_discord_notifications").
func notificationEventField(channel, event string) string {
	return event + "_" + channel + "_notifications"
}

func (r *notificationSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_settings"
}

func (r *notificationSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Team notification settings for one channel " +
			"(docs: knowledge-base/notifications). Singleton per channel: creating the resource " +
			"adopts the channel's settings, destroying it disables the channel. Event names for " +
			"`events` follow the Coolify UI: `deployment_success`, `deployment_failure`, " +
			"`status_change`, `backup_success`, `backup_failure`, `scheduled_task_success`, " +
			"`scheduled_task_failure`, `docker_cleanup_success`, `docker_cleanup_failure`, " +
			"`server_disk_usage`, `server_reachable`, `server_unreachable`, `server_patch`, " +
			"`traefik_outdated`.",
		Attributes: map[string]schema.Attribute{
			"channel": schema.StringAttribute{
				MarkdownDescription: "`email`, `discord`, `slack`, `telegram`, `pushover` or `webhook`. " +
					"Changing it forces replacement.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf(client.NotificationChannels...)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable the channel. Defaults to the current instance value.",
				Optional:            true,
				Computed:            true,
			},
			"webhook_url": schema.StringAttribute{
				MarkdownDescription: "Webhook URL (discord, slack and webhook channels).",
				Optional:            true,
				Sensitive:           true,
			},
			"telegram_token": schema.StringAttribute{
				MarkdownDescription: "Telegram bot token (telegram channel).",
				Optional:            true,
				Sensitive:           true,
			},
			"telegram_chat_id": schema.StringAttribute{
				MarkdownDescription: "Telegram chat id (telegram channel).",
				Optional:            true,
			},
			"pushover_user_key": schema.StringAttribute{
				MarkdownDescription: "Pushover user key (pushover channel).",
				Optional:            true,
				Sensitive:           true,
			},
			"pushover_api_token": schema.StringAttribute{
				MarkdownDescription: "Pushover API token (pushover channel).",
				Optional:            true,
				Sensitive:           true,
			},
			"smtp_host": schema.StringAttribute{
				MarkdownDescription: "SMTP host (email channel).",
				Optional:            true,
			},
			"smtp_port": schema.Int64Attribute{
				MarkdownDescription: "SMTP port (email channel).",
				Optional:            true,
			},
			"smtp_username": schema.StringAttribute{
				MarkdownDescription: "SMTP username (email channel).",
				Optional:            true,
			},
			"smtp_password": schema.StringAttribute{
				MarkdownDescription: "SMTP password (email channel).",
				Optional:            true,
				Sensitive:           true,
			},
			"smtp_from_address": schema.StringAttribute{
				MarkdownDescription: "Sender address (email channel).",
				Optional:            true,
			},
			"smtp_from_name": schema.StringAttribute{
				MarkdownDescription: "Sender display name (email channel).",
				Optional:            true,
			},
			"smtp_recipients": schema.StringAttribute{
				MarkdownDescription: "Comma-separated recipients (email channel).",
				Optional:            true,
			},
			"events": schema.MapAttribute{
				MarkdownDescription: "Per-event toggles for this channel (see the list above). " +
					"Only the listed keys are managed; others keep their instance values.",
				Optional:    true,
				ElementType: types.BoolType,
			},
		},
	}
}

func (r *notificationSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

// toBody builds the PATCH body from the configured (non-null) attributes only,
// so unmanaged fields keep their instance values.
func (r *notificationSettingsResource) toBody(ctx context.Context, m notificationSettingsResourceModel) (map[string]any, error) {
	channel := m.Channel.ValueString()
	body := map[string]any{}

	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		body[notificationEnabledField(channel)] = m.Enabled.ValueBool()
	}
	setString := func(field string, v types.String) {
		if !v.IsNull() && !v.IsUnknown() {
			body[field] = v.ValueString()
		}
	}
	setString(notificationWebhookField(channel), m.WebhookURL)
	setString("telegram_token", m.TelegramToken)
	setString("telegram_chat_id", m.TelegramChatID)
	setString("pushover_user_key", m.PushoverUserKey)
	setString("pushover_api_token", m.PushoverToken)
	setString("smtp_host", m.SMTPHost)
	setString("smtp_username", m.SMTPUsername)
	setString("smtp_password", m.SMTPPassword)
	setString("smtp_from_address", m.SMTPFromAddress)
	setString("smtp_from_name", m.SMTPFromName)
	setString("smtp_recipients", m.SMTPRecipients)
	if !m.SMTPPort.IsNull() && !m.SMTPPort.IsUnknown() {
		body["smtp_port"] = m.SMTPPort.ValueInt64()
	}

	if !m.Events.IsNull() && !m.Events.IsUnknown() {
		events := map[string]bool{}
		if diags := m.Events.ElementsAs(ctx, &events, false); diags.HasError() {
			return nil, fmt.Errorf("invalid events map")
		}
		for event, enabled := range events {
			body[notificationEventField(channel, event)] = enabled
		}
	}
	return body, nil
}

// refresh reads the channel and merges the API values into the model.
func (r *notificationSettingsResource) refresh(ctx context.Context, m notificationSettingsResourceModel) (notificationSettingsResourceModel, error) {
	channel := m.Channel.ValueString()
	settings, err := r.client.GetNotificationSettings(ctx, channel)
	if err != nil {
		return m, err
	}

	if v, ok := settings[notificationEnabledField(channel)].(bool); ok {
		m.Enabled = types.BoolValue(v)
	} else if m.Enabled.IsUnknown() {
		m.Enabled = types.BoolValue(false)
	}

	// Sensitive values are hidden without read:sensitive; configured wins.
	readString := func(field string, prior types.String) types.String {
		if v, ok := settings[field].(string); ok && v != "" {
			return types.StringValue(v)
		}
		return prior
	}
	m.WebhookURL = readString(notificationWebhookField(channel), m.WebhookURL)
	m.TelegramToken = readString("telegram_token", m.TelegramToken)
	m.TelegramChatID = readString("telegram_chat_id", m.TelegramChatID)
	m.PushoverUserKey = readString("pushover_user_key", m.PushoverUserKey)
	m.PushoverToken = readString("pushover_api_token", m.PushoverToken)
	m.SMTPHost = readString("smtp_host", m.SMTPHost)
	m.SMTPUsername = readString("smtp_username", m.SMTPUsername)
	m.SMTPPassword = readString("smtp_password", m.SMTPPassword)
	m.SMTPFromAddress = readString("smtp_from_address", m.SMTPFromAddress)
	m.SMTPFromName = readString("smtp_from_name", m.SMTPFromName)
	m.SMTPRecipients = readString("smtp_recipients", m.SMTPRecipients)
	if v, ok := settings["smtp_port"].(float64); ok {
		m.SMTPPort = types.Int64Value(int64(v))
	}

	// Only refresh the event keys the practitioner manages.
	if !m.Events.IsNull() && !m.Events.IsUnknown() {
		managed := map[string]bool{}
		if diags := m.Events.ElementsAs(ctx, &managed, false); !diags.HasError() {
			for event := range managed {
				if v, ok := settings[notificationEventField(channel, event)].(bool); ok {
					managed[event] = v
				}
			}
			events, diags := types.MapValueFrom(ctx, types.BoolType, managed)
			if !diags.HasError() {
				m.Events = events
			}
		}
	}
	return m, nil
}

func (r *notificationSettingsResource) apply(ctx context.Context, plan notificationSettingsResourceModel, diags *[]string) (notificationSettingsResourceModel, error) {
	body, err := r.toBody(ctx, plan)
	if err != nil {
		return plan, err
	}
	if len(body) > 0 {
		if _, err := r.client.UpdateNotificationSettings(ctx, plan.Channel.ValueString(), body); err != nil {
			return plan, err
		}
	}
	_ = diags
	return r.refresh(ctx, plan)
}

func (r *notificationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.apply(ctx, plan, nil)
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure Coolify notification channel", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *notificationSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.refresh(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify %s notification settings", state.Channel.ValueString()),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *notificationSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan notificationSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.apply(ctx, plan, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify %s notification settings", plan.Channel.ValueString()),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Delete disables the channel; the settings object itself is a singleton the
// API never deletes.
func (r *notificationSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state notificationSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channel := state.Channel.ValueString()
	_, err := r.client.UpdateNotificationSettings(ctx, channel, map[string]any{
		notificationEnabledField(channel): false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to disable Coolify %s notifications", channel),
			err.Error(),
		)
	}
}
