# Attach the same tag to several resources, then deploy them all at once
# with `coolify_deployment`'s `tag` attribute.
resource "coolify_resource_tag" "web" {
  resource_type = "application"
  resource_uuid = coolify_application.web.uuid
  tag_name      = "production"
}

resource "coolify_resource_tag" "worker" {
  resource_type = "application"
  resource_uuid = coolify_application.worker.uuid
  tag_name      = "production"
}

resource "coolify_resource_tag" "db" {
  resource_type = "database"
  resource_uuid = coolify_database.pg.uuid
  tag_name      = "production"
}
