resource "coolify_gitlab_app" "self_hosted" {
  name          = "gitlab-deployer"
  html_url      = "https://gitlab.example.com"
  group_name    = "platform"
  client_id     = var.gitlab_app_id
  client_secret = var.gitlab_app_secret
  webhook_token = var.gitlab_webhook_token
}
