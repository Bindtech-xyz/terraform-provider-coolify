# Restart the app whenever its config file changes.
resource "coolify_resource_action" "restart_web" {
  resource_type = "application"
  resource_uuid = coolify_application.web.uuid
  action        = "restart"

  triggers = {
    config_hash = sha1(coolify_storage.config.content)
  }
}
