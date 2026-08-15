resource "coolify_server_settings" "worker" {
  server_uuid = coolify_server.example.uuid

  docker_cleanup = {
    frequency             = "0 4 * * *"
    threshold             = 80
    delete_unused_volumes = false
  }

  sentinel = {
    enabled         = true
    metrics_enabled = true
  }
}
