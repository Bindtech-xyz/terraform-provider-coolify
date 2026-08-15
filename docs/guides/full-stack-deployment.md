---
page_title: "Deploying a Full Stack"
description: |-
  Wire a project, environment, database and application together, then scale the
  pattern to many applications with for_each.
---

# Deploying a Full Stack

A realistic deployment is rarely one resource. This guide builds a project with its
own environment, a PostgreSQL database, an application wired to it through an
environment variable, and a scheduled backup — then shows how the same shape scales to
many applications.

## The building blocks

```terraform
resource "coolify_project" "shop" {
  name = "shop"
}

resource "coolify_environment" "staging" {
  project_uuid = coolify_project.shop.uuid
  name         = "staging"
}

resource "coolify_database" "pg" {
  engine           = "postgresql"
  project_uuid     = coolify_project.shop.uuid
  environment_name = coolify_environment.staging.name
  server_uuid      = local.server_uuid

  name           = "shop-db"
  postgres_db    = "shop"
  instant_deploy = true
}

resource "coolify_application" "api" {
  project_uuid     = coolify_project.shop.uuid
  environment_name = coolify_environment.staging.name
  server_uuid      = local.server_uuid

  git_repository = "https://github.com/acme/shop-api"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
  instant_deploy = true
}

resource "coolify_environment_variable" "database_url" {
  parent_type = "application"
  parent_uuid = coolify_application.api.uuid
  key         = "DATABASE_URL"
  value       = coolify_database.pg.internal_db_url
}

resource "coolify_database_backup" "nightly" {
  database_uuid = coolify_database.pg.uuid
  frequency     = "@daily"
}
```

Three things worth noting:

- **`coolify_database.pg.internal_db_url` flows straight into an env var.** It is only
  populated in state if the token has `read:sensitive` (see
  [Token Abilities & Sensitive Data](authentication-and-token-abilities.html.md)) — with
  a `write`-only token this reference still works at apply time (Coolify resolves it
  server-side), but Terraform cannot show you the value in a plan.
- **`coolify_application` and `coolify_database` do not depend on each other** beyond
  the implicit reference through the env var — Terraform's graph handles ordering, no
  `depends_on` needed.
- **A dedicated environment isolates the whole stack** from anything else in the
  project; deleting `staging` later refuses while it still contains resources, which is
  Coolify's own safety net, not something this provider adds.

## Scaling out: many applications from one block

Once the shape above works for one app, `for_each` turns it into a fleet. Keep the
map keyed by something stable (a slug), not by index — reordering a list would
otherwise force-replace unrelated applications.

```terraform
locals {
  services = {
    api = {
      repo   = "acme/shop-api"
      branch = "main"
      port   = "3000"
    }
    worker = {
      repo   = "acme/shop-worker"
      branch = "main"
      port   = "8080"
    }
    admin = {
      repo   = "acme/shop-admin"
      branch = "main"
      port   = "3001"
    }
  }
}

resource "coolify_application" "fleet" {
  for_each = local.services

  project_uuid     = coolify_project.shop.uuid
  environment_name = coolify_environment.staging.name
  server_uuid      = local.server_uuid

  name           = each.key
  git_repository = "https://github.com/${each.value.repo}"
  git_branch     = each.value.branch
  build_pack     = "nixpacks"
  ports_exposes  = each.value.port
  instant_deploy = true
}

resource "coolify_environment_variable" "fleet_database_url" {
  for_each = local.services

  parent_type = "application"
  parent_uuid = coolify_application.fleet[each.key].uuid
  key         = "DATABASE_URL"
  value       = coolify_database.pg.internal_db_url
}
```

For variables that differ across the fleet, prefer `coolify_environment_variables`
(the bulk resource) over one `coolify_environment_variable` per key when a service has
more than a handful — it applies the whole map in a single API call and cleans up keys
removed from the map on the next apply:

```terraform
resource "coolify_environment_variables" "fleet_config" {
  for_each = local.services

  parent_type = "application"
  parent_uuid = coolify_application.fleet[each.key].uuid

  variables = {
    DATABASE_URL = coolify_database.pg.internal_db_url
    APP_ENV      = "staging"
    SERVICE_NAME = each.key
  }
}
```

## Next steps

- [Provisioning Cloud Servers](cloud-provisioning.html.md) if the fleet needs its own
  dedicated VMs rather than sharing one server.
- [The Dynamic Service Catalog](dynamic-service-catalog.html.md) to add one-click
  services (databases admin UIs, monitoring, …) alongside the fleet.
