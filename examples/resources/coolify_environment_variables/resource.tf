# Bulk-manage a whole set of variables in one call.
resource "coolify_environment_variables" "web" {
  parent_type = "application"
  parent_uuid = coolify_application.web.uuid

  variables = {
    DATABASE_URL = coolify_database.pg.internal_db_url
    REDIS_URL    = coolify_database.cache.internal_db_url
    APP_ENV      = "production"
  }
}
