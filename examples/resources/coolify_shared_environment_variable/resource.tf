resource "coolify_shared_environment_variable" "team_wide" {
  scope = "team"
  key   = "SENTRY_DSN"
  value = "https://..."
}

# Usable in resources as {{environment.API_URL}}
resource "coolify_shared_environment_variable" "per_env" {
  scope        = "environment"
  project_uuid = coolify_project.example.uuid
  environment  = "production"
  key          = "API_URL"
  value        = "https://api.example.com"
}
