resource "coolify_destination" "isolated" {
  server_uuid = coolify_server.example.uuid
  network     = "isolated-net"
}
