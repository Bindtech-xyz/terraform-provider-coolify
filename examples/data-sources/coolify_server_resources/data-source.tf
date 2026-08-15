data "coolify_server_resources" "main" {
  server_uuid = data.coolify_server.main.uuid
}

output "running_workloads" {
  value = [for r in data.coolify_server_resources.main.resources : r.name
  if startswith(r.status, "running")]
}
