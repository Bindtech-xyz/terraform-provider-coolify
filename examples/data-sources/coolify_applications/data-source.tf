data "coolify_applications" "all" {}

output "running" {
  value = [for a in data.coolify_applications.all.applications : a.name
  if startswith(a.status, "running")]
}
