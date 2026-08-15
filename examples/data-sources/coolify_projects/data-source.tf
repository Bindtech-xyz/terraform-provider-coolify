data "coolify_projects" "all" {}

output "project_uuids" {
  value = data.coolify_projects.all.projects[*].uuid
}
