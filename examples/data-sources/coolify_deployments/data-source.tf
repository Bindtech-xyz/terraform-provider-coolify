# Currently queued/running deployments on the instance.
data "coolify_deployments" "running" {}

# History of one application.
data "coolify_deployments" "web" {
  application_uuid = coolify_application.web.uuid
}
