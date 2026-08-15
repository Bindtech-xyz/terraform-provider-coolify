# The API has no read endpoint for volume backup schedules (PUT-only upsert),
# so importing only establishes which storage the schedule belongs to — its
# attributes (frequency, retention, ...) are populated by the next apply.
terraform import coolify_volume_backup.example <parent_type>/<parent_uuid>/<storage_uuid>

# parent_type is "application", "service" or "database".
