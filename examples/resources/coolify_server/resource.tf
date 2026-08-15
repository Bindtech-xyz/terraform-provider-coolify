# The private key must already exist in Coolify (Keys & Tokens → Private Keys).
resource "coolify_server" "example" {
  name             = "worker-01"
  ip               = "203.0.113.10"
  port             = 22
  user             = "root"
  private_key_uuid = "sk-uuid-from-coolify"
  proxy_type       = "traefik"
}
