# Attach a second standalone Docker destination to an application, on a
# different server than its primary destination.
resource "coolify_application_destination" "web_secondary" {
  application_uuid = coolify_application.web.uuid
  destination_uuid = data.coolify_destinations.secondary.destinations[0].uuid
}
