data "coolify_server_domains" "main" {
  server_uuid = data.coolify_server.main.uuid
}
