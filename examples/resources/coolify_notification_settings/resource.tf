resource "coolify_notification_settings" "discord" {
  channel     = "discord"
  enabled     = true
  webhook_url = var.discord_webhook

  events = {
    deployment_failure = true
    backup_failure     = true
    server_unreachable = true
  }
}
