resource "coolify_volume_backup" "uploads" {
  parent_type     = "application"
  parent_uuid     = coolify_application.web.uuid
  storage_uuid    = coolify_storage.uploads.uuid
  frequency       = "@daily"
  save_s3         = true
  s3_storage_uuid = coolify_s3_storage.backups.uuid
}
