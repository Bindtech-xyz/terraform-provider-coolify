data "coolify_project" "example" {
  uuid = "og888os"
}

output "project_name" {
  value = data.coolify_project.example.name
}
