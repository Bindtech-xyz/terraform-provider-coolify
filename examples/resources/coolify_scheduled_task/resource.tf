resource "coolify_scheduled_task" "cleanup" {
  parent_type = "application"
  parent_uuid = coolify_application.web.uuid
  name        = "nightly-cleanup"
  command     = "php artisan cache:clear"
  frequency   = "0 3 * * *"
}
