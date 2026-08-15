# Provisions the VM at Hetzner AND registers it as a Coolify server.
resource "coolify_cloud_server" "worker" {
  provider_name    = "hetzner"
  cloud_token_uuid = coolify_cloud_token.hetzner.uuid
  private_key_uuid = coolify_private_key.deploy.uuid

  name             = "worker-02"
  location         = "fsn1"
  server_type      = "cx32"
  hetzner_image_id = 114690389 # ubuntu-24.04
}
