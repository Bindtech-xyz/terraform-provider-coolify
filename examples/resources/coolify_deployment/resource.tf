# Deploy a single application and block until it finishes.
resource "coolify_deployment" "web" {
  resource_uuid       = coolify_application.web.uuid
  wait_for_completion = true
  timeout_seconds     = 900

  triggers = {
    image_tag = var.image_tag
  }
}

# Deploy every resource carrying the "production" tag whenever it changes.
resource "coolify_resource_tag" "web_prod" {
  resource_type = "application"
  resource_uuid = coolify_application.web.uuid
  tag_name      = "production"
}

resource "coolify_deployment" "by_tag" {
  tag = "production"

  triggers = {
    run_id = timestamp()
  }
}
