data "coolify_cloud_catalog" "hetzner_locations" {
  provider_name    = "hetzner"
  section          = "locations"
  cloud_token_uuid = coolify_cloud_token.hetzner.uuid
}

output "locations" {
  value = data.coolify_cloud_catalog.hetzner_locations.names
}
