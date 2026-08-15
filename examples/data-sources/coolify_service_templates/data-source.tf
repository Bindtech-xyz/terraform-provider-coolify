# The catalog is fetched live from the official CDN feed — it always matches
# what Coolify itself offers, without provider updates.
data "coolify_service_templates" "all" {}

# Validate a type before using it:
locals {
  wanted = "plausible"
  valid  = contains([for t in data.coolify_service_templates.all.types : t], local.wanted)
}

# Or browse one category only:
data "coolify_service_templates" "analytics" {
  category = "analytics"
}
