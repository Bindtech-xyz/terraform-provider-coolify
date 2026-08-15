resource "coolify_cloud_token" "hetzner" {
  name          = "hetzner-prod"
  provider_name = "hetzner"
  token         = var.hetzner_api_token
}
