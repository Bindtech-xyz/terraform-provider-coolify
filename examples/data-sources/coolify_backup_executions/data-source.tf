data "coolify_backup_executions" "pg_nightly" {
  database_uuid = coolify_database.pg.uuid
  backup_uuid   = coolify_database_backup.nightly.uuid
}

output "last_backup_status" {
  value = try(data.coolify_backup_executions.pg_nightly.executions[0].status, "none")
}
