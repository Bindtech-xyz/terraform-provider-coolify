resource "coolify_database" "pg" {
  engine           = "postgresql"
  project_uuid     = coolify_project.example.uuid
  environment_name = "production"
  server_uuid      = coolify_server.example.uuid

  name        = "app-db"
  postgres_db = "app"

  instant_deploy = true
}

output "database_url" {
  value     = coolify_database.pg.internal_db_url
  sensitive = true
}
