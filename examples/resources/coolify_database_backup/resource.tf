resource "coolify_database_backup" "nightly" {
  database_uuid   = coolify_database.pg.uuid
  frequency       = "@daily"
  save_s3         = true
  s3_storage_uuid = coolify_s3_storage.backups.uuid

  retention_amount_locally = 3
  retention_days_s3        = 30
}
