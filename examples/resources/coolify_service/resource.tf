# One-click service from the dynamic catalog.
resource "coolify_service" "analytics" {
  type             = "plausible"
  project_uuid     = coolify_project.example.uuid
  environment_name = "production"
  server_uuid      = coolify_server.example.uuid
  instant_deploy   = true
}

# Custom docker-compose service.
resource "coolify_service" "custom" {
  docker_compose_raw = base64encode(file("${path.module}/docker-compose.yml"))
  project_uuid       = coolify_project.example.uuid
  environment_name   = "production"
  server_uuid        = coolify_server.example.uuid
}
