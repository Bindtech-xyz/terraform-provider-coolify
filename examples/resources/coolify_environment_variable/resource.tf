resource "coolify_environment_variable" "database_url" {
  parent_type = "application"
  parent_uuid = coolify_application.web.uuid
  key         = "DATABASE_URL"
  value       = coolify_database.pg.internal_db_url
}
