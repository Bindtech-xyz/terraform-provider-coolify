data "coolify_servers" "all" {}

output "usable_servers" {
  value = [for s in data.coolify_servers.all.servers : s.uuid if s.is_usable]
}
