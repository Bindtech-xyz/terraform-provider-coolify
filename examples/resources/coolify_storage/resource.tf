# Named volume. Coolify prefixes the volume name with the parent's UUID
# server-side — the real, effective name is exposed as volume_name.
resource "coolify_storage" "uploads" {
  parent_type = "application"
  parent_uuid = coolify_application.web.uuid
  type        = "persistent"
  name        = "uploads"
  mount_path  = "/app/uploads"
}

output "uploads_volume_name" {
  value = coolify_storage.uploads.volume_name # e.g. "abc123-uploads"
}

# Inline config file mount.
resource "coolify_storage" "config" {
  parent_type = "application"
  parent_uuid = coolify_application.web.uuid
  type        = "file"
  mount_path  = "/app/config.yml"
  content     = file("${path.module}/config.yml")
}
