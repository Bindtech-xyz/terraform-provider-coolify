resource "coolify_github_app" "org" {
  name             = "coolify-deployer"
  organization     = "acme"
  app_id           = 123456
  installation_id  = 654321
  client_id        = "Iv1.abc123"
  client_secret    = var.github_app_client_secret
  webhook_secret   = var.github_app_webhook_secret
  private_key_uuid = coolify_private_key.github_app.uuid
}

# Then deploy private repos with it:
# resource "coolify_application" "private" {
#   github_app_uuid = coolify_github_app.org.uuid
#   git_repository  = "acme/private-api"
#   ...
# }
