# Terraform Provider for Coolify

A [Terraform](https://terraform.io) / [OpenTofu](https://opentofu.org) provider for
[Coolify](https://coolify.io) v4 — built to manage a whole instance and deploy fleets of
applications as code. Written against the Coolify **main branch** API (v4.3.x controllers,
not the lagging OpenAPI spec), on the
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
(protocol v6, Terraform ≥ 1.1).

## Coverage

| Object | Resource | Data source |
| --- | :--: | :--: |
| Private keys | `coolify_private_key` | `coolify_private_keys` |
| Servers | `coolify_server` | `coolify_servers` |
| Destinations (Docker networks) | `coolify_destination` | — |
| Projects | `coolify_project` | `coolify_project`, `coolify_projects` |
| Environments | `coolify_environment` | `coolify_environments` |
| Applications (5 modes: public git, deploy key, GitHub App, Dockerfile, registry image) | `coolify_application` | `coolify_applications` |
| Databases (8 engines: postgresql, mysql, mariadb, mongodb, redis, keydb, dragonfly, clickhouse) | `coolify_database` | `coolify_databases` |
| Services (one-click **and** raw docker-compose) | `coolify_service` | `coolify_services` |
| Service catalog (**dynamic**, 300+ templates from the live CDN feed) | — | `coolify_service_templates` |
| Env vars (unitary + **bulk**) | `coolify_environment_variable`, `coolify_environment_variables` | — |
| Shared env vars (team/project/environment/server) | `coolify_shared_environment_variable` | — |
| Persistent storage (volumes & file mounts) | `coolify_storage` | — |
| Volume backups | `coolify_volume_backup` | — |
| Scheduled tasks (cron) | `coolify_scheduled_task` | — |
| Database backups (+ S3, retention) | `coolify_database_backup` | `coolify_backup_executions` |
| S3 storages | `coolify_s3_storage` | `coolify_s3_storages` |
| Notifications (email/discord/slack/telegram/pushover/webhook) | `coolify_notification_settings` | — |
| Server settings (proxy, docker cleanup, Sentinel, Cloudflare Tunnel, **log drains**) | `coolify_server_settings` | `coolify_server_domains`, `coolify_server_resources` |
| GitHub / GitLab Apps (private repos CI/CD) | `coolify_github_app`, `coolify_gitlab_app` | `coolify_github_app_repositories` |
| Cloud provisioning (Hetzner/DigitalOcean/Vultr) | `coolify_cloud_server`, `coolify_cloud_token`, `coolify_cloud_init_script` | `coolify_cloud_catalog` |
| Start/stop/restart (declarative trigger) | `coolify_resource_action` | — |
| Deployments | — | `coolify_deployments` |
| Instance (health, version) | — | `coolify_instance` |
| Tags | `coolify_tag` | `coolify_tags` |
| Teams | — | `coolify_team`, `coolify_teams` |

**25 resources, 22 data sources.** Deletes of applications/databases/services/destinations
poll until Coolify's asynchronous teardown actually finishes (backoff 500ms→5s), so
destroy-then-recreate cycles never collide on names, domains or networks.

Coverage tracks the [Coolify docs sidebar](https://coolify.io/docs) — every
API-manageable concept in Applications, Databases, Services, Knowledge Base
(destinations, S3, env vars, persistent storage, cron, notifications, server settings)
and Integrations (Cloudflare Tunnel) has a resource or data source. This also covers
the gaps left by the existing community providers.

## Usage

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "d3nailabs/coolify"
      version = "~> 0.1"
    }
  }
}

provider "coolify" {
  endpoint = "https://coolify.example.com" # defaults to Coolify Cloud
  # token via the COOLIFY_TOKEN environment variable (Keys & Tokens → API tokens)
}
```

Full stack in one apply — project, environment, database, app wired together:

```hcl
resource "coolify_project" "shop" {
  name = "shop"
}

resource "coolify_database" "pg" {
  engine           = "postgresql"
  project_uuid     = coolify_project.shop.uuid
  environment_name = "production"
  server_uuid      = data.coolify_servers.all.servers[0].uuid
  instant_deploy   = true
}

resource "coolify_application" "api" {
  project_uuid     = coolify_project.shop.uuid
  environment_name = "production"
  server_uuid      = data.coolify_servers.all.servers[0].uuid

  git_repository = "https://github.com/acme/shop-api"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
  instant_deploy = true
}

resource "coolify_environment_variable" "db" {
  parent_type = "application"
  parent_uuid = coolify_application.api.uuid
  key         = "DATABASE_URL"
  value       = coolify_database.pg.internal_db_url
}
```

Deploying many apps is a `for_each` away — see `examples/`.

### API token abilities

Create the token with the `write` ability (and `read:sensitive` if you want generated
database credentials and env-var values readable back into state — without it the
provider keeps your configured values and never diffs on hidden ones).

## Development

Requirements: Go ≥ 1.25, Terraform ≥ 1.1 (or OpenTofu), GNU make.

```sh
make build     # compile
make test      # unit tests (httptest, no credentials needed)
make install   # go install into GOBIN for dev_overrides
make testacc   # acceptance tests — creates real objects, use a disposable instance!
make docs      # regenerate docs/ with tfplugindocs
```

### Local iteration with dev_overrides

No local registry, no version bumps, no lock-file fights — put this in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "d3nailabs/coolify" = "/home/<you>/go/bin"
  }
  direct {}
}
```

Then `make install` and run `terraform plan`/`apply` directly — **skip `terraform init`**,
dev_overrides bypasses it.

### Acceptance tests

```sh
export COOLIFY_ENDPOINT="https://coolify.example.com"
export COOLIFY_TOKEN="..."
TF_ACC=1 go test ./internal/provider/ -v -run TestAccProjectResource
```

`TestAccApplicationResource` additionally deploys a real container (`nginx:alpine`,
no build) end to end — creates it, polls until Coolify reports it `running:*`, then
destroys it. It needs `COOLIFY_ACC_SERVER_UUID` (an existing, usable server with a
single destination) and is skipped, not failed, when that variable is unset — most
environments running the rest of the suite have no deployable server available.

If the instance sits behind an authenticating edge (see the
[Reaching Coolify Behind an Authenticating Edge](https://registry.terraform.io/providers/d3nailabs/coolify/latest/docs/guides/reverse-proxy-authentication)
guide), also set `CF_ACCESS_CLIENT_ID`/`CF_ACCESS_CLIENT_SECRET` (or the equivalent for
your edge layer) — every acceptance test's `Config` prepends a `provider "coolify" {}`
block carrying them as `headers` when both are set.

## Releasing

1. Register your GPG public key on registry.terraform.io (User Settings → Signing Keys).
2. Set the `GPG_PRIVATE_KEY` and `PASSPHRASE` repository secrets.
3. `git tag v0.1.0 && git push origin v0.1.0` — the Release workflow runs GoReleaser and
   the registry picks the release up via webhook (first publication is manual in the
   registry UI: Publish → Provider).

## Not covered (and why)

Preview deployments have no list/read API (only a delete endpoint) so they cannot be
managed declaratively; server transfer/claim endpoints are interactive workflows. Note
that `coolify_cloud_server` destroy only deregisters the server from Coolify — the VM
itself must be cleaned up at the provider.
