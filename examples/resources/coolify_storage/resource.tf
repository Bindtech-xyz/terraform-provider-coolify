# Named volume.
resource "coolify_storage" "uploads" {
  parent_type = "application"
  parent_uuid = coolify_application.web.uuid
  type        = "persistent"
  name        = "uploads"
  mount_path  = "/app/uploads"
}

# Inline config file mount.
resource "coolify_storage" "config" {
  parent_type = "application"
  parent_uuid = coolify_application.web.uuid
  type        = "file"
  mount_path  = "/app/config.yml"
  content     = file("${path.module}/config.yml")
}
