---
page_title: "Importing Existing Infrastructure"
description: |-
  The terraform import ID format for every importable resource, including the
  composite ones and the deliberate exceptions.
---

# Importing Existing Infrastructure

Most resources import by UUID alone. A third of them need a **composite ID** —
because Coolify addresses the object by more than one coordinate (a variable inside a
scope, a storage inside a resource, an engine-ambiguous database) — and getting the
format wrong is a confusing way to find out. This page is the single source of truth
for every resource's import syntax.

## Simple: UUID only

```sh
terraform import coolify_project.example <uuid>
```

The same one-argument form applies to: `coolify_application`, `coolify_server`,
`coolify_private_key`, `coolify_destination`, `coolify_tag`, `coolify_service`,
`coolify_s3_storage`, `coolify_cloud_token`, `coolify_cloud_init_script`,
`coolify_cloud_server`.

## Simple: numeric id, not UUID

GitHub and GitLab sources are the only resources Coolify addresses by a plain
incrementing integer instead of a UUID — copy the id shown in the Coolify UI or from
`GET /github-apps` / `GET /gitlab-apps`:

```sh
terraform import coolify_github_app.example <numeric-id>
terraform import coolify_gitlab_app.example <numeric-id>
```

## Simple: a different attribute than `uuid`

Two resources import by an attribute other than `uuid`:

```sh
# coolify_server_settings tracks a server, not itself — only server_uuid is
# populated on import; configure the blocks (proxy, docker_cleanup, sentinel,
# cloudflare_tunnel, log_drains) you want managed, then apply.
terraform import coolify_server_settings.example <server_uuid>

# coolify_notification_settings is a singleton per channel.
terraform import coolify_notification_settings.discord discord
# one of: email, discord, slack, telegram, pushover, webhook
```

## Composite: `<parent_type>/<parent_uuid>/<child>`

Four resources are scoped to a parent resource (an application, service or database)
and need that parent identified alongside the child's own id:

```sh
terraform import coolify_environment_variable.example application/<parent_uuid>/<key>
terraform import coolify_storage.example application/<parent_uuid>/<storage_uuid>
terraform import coolify_scheduled_task.example application/<parent_uuid>/<task_uuid>
terraform import coolify_volume_backup.example application/<parent_uuid>/<storage_uuid>
```

`parent_type` is `application` or `service` for `coolify_scheduled_task`; add
`database` for the other three (databases have storages and env vars, but no scheduled
tasks).

`coolify_volume_backup` has one more wrinkle: **Coolify's API has no read endpoint for
volume backup schedules** (it is a write-only upsert). Import only establishes which
storage the schedule belongs to — the schedule's own attributes (`frequency`,
`retention_*`, …) come from your configuration on the very next `apply`, not from
reading the current schedule. If your configuration does not yet match what is actually
scheduled, that apply will silently overwrite it rather than show a diff to review.
Check the schedule in the Coolify UI before importing this one.

## Composite: two independent UUIDs

```sh
terraform import coolify_database_backup.example <database_uuid>/<backup_uuid>
```

## Composite: engine prefix

```sh
terraform import coolify_database.example postgresql/<uuid>
# one of: postgresql, mysql, mariadb, mongodb, redis, keydb, dragonfly, clickhouse
```

Coolify's database object does not carry its engine in a form the provider can reuse
directly — `redis`, `keydb` and `dragonfly` all speak the same wire protocol, so even
inspecting the running container wouldn't disambiguate reliably. `engine` is
`Required` with `RequiresReplace`; importing with the wrong engine does not fail
immediately, but the first `plan` afterward proposes destroying and recreating the
database against the correct engine's create endpoint.

## Composite: scope-dependent format

`coolify_shared_environment_variable` has four distinct forms depending on `scope`:

```sh
terraform import coolify_shared_environment_variable.team_wide team/<key>
terraform import coolify_shared_environment_variable.per_project project/<project_uuid>/<key>
terraform import coolify_shared_environment_variable.per_env environment/<project_uuid>/<environment>/<key>
terraform import coolify_shared_environment_variable.per_server server/<server_uuid>/<key>
```

## Not importable, by design

Two resources have no `terraform import` support — not an oversight, but because
neither maps to a single existing object with a stable identity to import:

| Resource | Why |
| --- | --- |
| `coolify_environment_variables` | Manages an arbitrary *subset* of a parent's variables (a map you choose). There is no single "the" set of variables to import — bring individual keys under management with `coolify_environment_variable` instead, one `terraform import` per key. |
| `coolify_resource_action` | Fire-and-forget: it triggers a start/stop/restart and holds no state that corresponds to anything on the Coolify side afterward. There is nothing to import. |

Every other resource in the provider supports import — if one you expected to see
above is missing, that is a documentation bug worth reporting.
