data "coolify_team" "current" {}

output "team_name" {
  value = data.coolify_team.current.name
}
