# Public git repository built with nixpacks.
resource "coolify_application" "web" {
  project_uuid     = coolify_project.example.uuid
  environment_name = "production"
  server_uuid      = coolify_server.example.uuid

  git_repository = "https://github.com/coollabsio/coolify-examples"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"

  domains        = "https://web.example.com"
  instant_deploy = true
}

# Image from a registry.
resource "coolify_application" "nginx" {
  project_uuid     = coolify_project.example.uuid
  environment_name = "production"
  server_uuid      = coolify_server.example.uuid

  docker_registry_image_name = "nginx"
  docker_registry_image_tag  = "1.27-alpine"
  ports_exposes              = "80"
}
