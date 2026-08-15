resource "coolify_environment" "staging" {
  project_uuid = coolify_project.example.uuid
  name         = "staging"
}
