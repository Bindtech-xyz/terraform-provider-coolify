---
page_title: "The Dynamic Service Catalog"
description: |-
  Discover and validate one-click service types without waiting on a provider release.
---

# The Dynamic Service Catalog

`coolify_service.type` picks a one-click service — `plausible`, `gitea`, `n8n`, and
[300+ more](https://coolify.io/docs/services/all). This provider does not hardcode
that list as a validated enum, and that is a deliberate choice worth explaining.

## Why not just validate `type` against a fixed list?

Coolify's catalog is a CDN-hosted JSON feed
(`https://cdn.coollabs.io/coolify/service-templates-latest.json`) that Coolify itself
fetches at runtime — it is not compiled into Coolify's own releases either. It grows,
shrinks and reshuffles between Coolify versions, sometimes within days: entries get
renamed, deprecated ones drop out, new ones appear constantly. A `stringvalidator.OneOf`
baked into this provider at build time would go stale the moment the catalog changed
underneath it — either rejecting a `type` that Coolify would happily accept, or (worse)
accepting one Coolify no longer recognizes, deferring the failure to `apply` time with a
less clear error.

Instead, `coolify_service_templates` fetches the same feed and exposes it as a data
source, so `type` validity is checked against **live** catalog state, on every `plan`
you choose to check it — never against a snapshot frozen at provider-release time.

## Discovering what's available

```terraform
data "coolify_service_templates" "all" {}

output "service_count" {
  value = length(data.coolify_service_templates.all.types)
}

output "analytics_options" {
  value = [for t, meta in data.coolify_service_templates.all.templates : t if meta.category == "analytics"]
}
```

Filter server-side by category instead of fetching everything when you already know
what you're looking for:

```terraform
data "coolify_service_templates" "analytics" {
  category = "analytics"
}
```

Each entry carries `slogan`, `category`, `documentation` (a link to the service's own
docs) and `port` (the default exposed port) — enough to build a self-service picker UI
or a validated variable without maintaining a second copy of Coolify's catalog by hand.

## Validating `type` before you deploy

```terraform
variable "service_type" {
  type = string
}

data "coolify_service_templates" "all" {}

locals {
  # Fails the plan with a readable error instead of a confusing 422 from Coolify.
  valid_service_type = contains(data.coolify_service_templates.all.types, var.service_type) ? var.service_type : file("ERROR: '${var.service_type}' is not a known Coolify service type")
}

resource "coolify_service" "app" {
  type              = local.valid_service_type
  project_uuid      = coolify_project.example.uuid
  environment_name  = "production"
  server_uuid       = local.server_uuid
  instant_deploy    = true
}
```

The `file(...)` trick forces a plan-time error with your message instead of a runtime
one — a common pattern for turning a `contains()` check into a hard failure without
reaching for `precondition` blocks (which need Terraform ≥ 1.2 and a bit more
ceremony for a one-off check like this).

## Overriding the feed URL

Self-hosted or air-gapped instances can point at a mirror instead of the public CDN:

```terraform
data "coolify_service_templates" "all" {
  url = "https://internal-mirror.example.com/service-templates.json"
}
```

The URL must serve the same shape Coolify's own feed does: a JSON object keyed by
service type, each value carrying at least `slogan`, `category`, `documentation` and
`port`.
