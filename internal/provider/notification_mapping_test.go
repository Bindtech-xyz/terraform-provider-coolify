package provider

import "testing"

// The notification API fields are derived per channel; these mappings are the
// contract between the consolidated resource and the six Laravel models.
func TestNotificationEnabledField(t *testing.T) {
	cases := map[string]string{
		"email":    "smtp_enabled", // email's toggle is smtp_enabled, not email_enabled
		"discord":  "discord_enabled",
		"slack":    "slack_enabled",
		"telegram": "telegram_enabled",
		"pushover": "pushover_enabled",
		"webhook":  "webhook_enabled",
	}
	for channel, want := range cases {
		if got := notificationEnabledField(channel); got != want {
			t.Errorf("enabled field for %s = %q, want %q", channel, got, want)
		}
	}
}

func TestNotificationWebhookField(t *testing.T) {
	cases := map[string]string{
		"discord": "discord_webhook_url",
		"slack":   "slack_webhook_url",
		"webhook": "webhook_url", // the webhook channel has no prefix
	}
	for channel, want := range cases {
		if got := notificationWebhookField(channel); got != want {
			t.Errorf("webhook field for %s = %q, want %q", channel, got, want)
		}
	}
}

func TestNotificationEventField(t *testing.T) {
	if got := notificationEventField("discord", "deployment_failure"); got != "deployment_failure_discord_notifications" {
		t.Errorf("event field = %q", got)
	}
	if got := notificationEventField("email", "backup_success"); got != "backup_success_email_notifications" {
		t.Errorf("event field = %q", got)
	}
}
