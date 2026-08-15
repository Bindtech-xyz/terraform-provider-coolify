# Coolify creates the preview itself from a GitHub App pull-request webhook
# — this resource does not provision it, only guarantees it's torn down
# when destroyed (e.g. as part of a CI job that closes out a PR's stack).
resource "coolify_application_preview" "pr" {
  application_uuid = coolify_application.web.uuid
  pull_request_id  = 42
}
