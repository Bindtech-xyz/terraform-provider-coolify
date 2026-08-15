# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versioning follows
[Semantic Versioning](https://semver.org/) once the first tag is cut. Nothing has been
released yet — everything below is `[Unreleased]`.

## [Unreleased]

### Added

- Provider-level `headers` attribute: a generic, sensitive map of fixed HTTP headers
  sent with every request, applied before (and unable to override) the provider's own
  `Authorization` header. Not modeled after any specific reverse proxy — it exists for
  reaching a Coolify instance behind any authenticating edge layer (Cloudflare Access
  service tokens, `oauth2-proxy`, a header-based mTLS gateway, ...). Unknown values
  (e.g. an edge-auth resource's attributes not yet applied) produce an explicit
  "provider block cannot depend on a same-apply resource" diagnostic instead of an
  opaque connectivity failure. See the
  [Reaching Coolify Behind an Authenticating Edge](https://registry.terraform.io/providers/d3nailabs/coolify/latest/docs/guides/reverse-proxy-authentication)
  guide.
- Initial provider: 25 resources and 22 data sources on `terraform-plugin-framework`
  v1 (protocol v6), covering the full [Coolify documentation](https://coolify.io/docs)
  sidebar — servers, projects, environments, all 5 application source modes, all 8
  database engines, one-click and custom-compose services, persistent storage,
  scheduled tasks, backups, notifications, GitHub/GitLab sources, and
  Hetzner/DigitalOcean/Vultr provisioning.
- `coolify_service_templates` data source: fetches Coolify's live one-click service
  catalog (300+ entries) from the CDN feed Coolify itself uses, so `coolify_service.type`
  is never validated against a list frozen at provider-build time.
- `coolify_application` consolidates all 5 Coolify create endpoints (public git,
  private git via deploy key or GitHub App, inline Dockerfile, registry image) into one
  resource, discriminated by which source attributes are configured.
- `coolify_database` consolidates all 8 engine-specific create endpoints (postgresql,
  mysql, mariadb, mongodb, redis, keydb, dragonfly, clickhouse) into one resource with
  an `engine` attribute.
- `coolify_cloud_server` provisions a VM at Hetzner, DigitalOcean or Vultr and
  registers it with Coolify in one apply, alongside `coolify_cloud_token`,
  `coolify_cloud_init_script` and the `coolify_cloud_catalog` data source for
  discovering valid regions/sizes/images per provider.
- Async-delete handling: deleting an application, database, service or destination
  polls (500ms → 5s backoff) until Coolify's asynchronous teardown actually finishes,
  so a destroy immediately followed by a create of the same name never races a
  container still being torn down.
- Sensitive-field handling that works with or without the API token's `read:sensitive`
  ability: generated database credentials, private key material, S3 keys and env-var
  values keep their last-known state value instead of being overwritten by the empty
  string Coolify's API returns when that ability is absent — see the
  [Token Abilities & Sensitive Data](https://registry.terraform.io/providers/d3nailabs/coolify/latest/docs/guides/authentication-and-token-abilities)
  guide.
- Composite import support for every resource whose identity spans more than one
  coordinate (`coolify_environment`, `coolify_storage`, `coolify_scheduled_task`,
  `coolify_database_backup`, `coolify_volume_backup`, `coolify_shared_environment_variable`,
  and `coolify_database`'s `<engine>/<uuid>` form) — documented end to end in the
  [Importing Existing Infrastructure](https://registry.terraform.io/providers/d3nailabs/coolify/latest/docs/guides/importing-existing-infrastructure)
  guide.
- `coolify_resource_action` for declarative start/stop/restart triggered by a
  `triggers` map, and `coolify_environment_variables` for bulk-managing a whole set of
  environment variables in one call.
- Forgejo Actions CI (`.forgejo/workflows/`) alongside a GitHub Actions mirror, and a
  GoReleaser pipeline producing Terraform-Registry-compliant release assets for both
  GitHub and Forgejo targets.
- Registry documentation: per-resource/data-source subcategories for left-nav
  grouping, six narrative guides under `docs/guides/`, and a rich provider overview
  page covering authentication and the design choices above.

### Fixed

- `coolify_database`: `limits_memory`/`limits_cpus` were write-only — the client's
  `Database` struct didn't even parse them back from `GET`, so a configured limit was
  applied once at create and never verified again (silent, undetectable drift). Added
  the fields to the client struct, marked both `Optional`+`Computed` with
  `UseStateForUnknown` (same fix as `coolify_application`'s limits). Verified against
  a real `postgresql` database: configured `256m`/`0.5` round-trips exactly, confirmed
  both in Terraform state and via a direct API call.
- `coolify_private_key.description`: `Optional`-only, but Coolify defaults it to
  `"Created by Coolify via API"` when unset (confirmed in the Laravel controller) —
  the same "provider produced inconsistent result" crash as the `coolify_application`
  fields above, on the very first create with no description set. A scan of every API
  controller found this exact auto-fill pattern nowhere else. Now `Optional`+`Computed`.
- `coolify_storage`: a deeper issue than a missing `Computed` flag — the resource's
  entire response-mapping model was wrong. Coolify has no shared "storage" concept:
  `persistent` and `file` mounts are two separate Eloquent models
  (`LocalPersistentVolume`, `LocalFileVolume`) with no discriminator column, so
  (a) `type` has **no corresponding field in any response** (the client's
  `json:"storage_type"` tag decoded to `""` always, not "sometimes empty" — it was
  simply wrong from the start), (b) `name` comes back **prefixed with the parent's
  UUID** server-side (`<parent-uuid>-<name>`), not the configured value, and
  (c) `content` is unconditionally `$hidden` in the Laravel model — the API never
  returns it, independent of the token's `read:sensitive` ability. Adopting any of
  these from the API produced a crash (`name`) or a silently wrong value (`type`
  always `""`, `content` always `""` instead of preserving the configured value — a
  latent bug in the existing null-preservation helper, exposed by the first real
  create with `content` unset). Fixed by adopting **only** `uuid` from the API;
  `type`/`name`/`content`/`mount_path`/`host_path` are now always echoed from
  config/prior state, which is exact by construction (`type` is `RequiresReplace`
  anyway). Added a new `volume_name` computed-only attribute exposing the real,
  UUID-prefixed name Coolify assigns — genuinely useful information that was simply
  going nowhere before. Separately, `GET .../storages` does not return a flat array
  either: the response is `{"persistent_storages": [...], "file_storages": [...]}`,
  which `ListStorages` now merges instead of failing to unmarshal outright.
- `coolify_application`: seven attributes (`base_directory`, `limits_memory`,
  `limits_cpus`, `git_branch`, `git_commit_sha`, `build_pack`, `static_image`) were
  `Optional`-only, but Coolify assigns each a non-empty default whenever it is left
  unset (`base_directory` → `/`, the limits → `0`, `git_branch` → `main`,
  `git_commit_sha` → `HEAD`, `build_pack`/`static_image` → the effective build
  strategy/image) — never `""`. An `Optional`-only attribute whose final state
  diverges from a planned `null` value is a framework-level "provider produced
  inconsistent result" error, which aborted `apply` for essentially every
  non-git-mode application (found by deploying a real `dockerimage`-mode
  application end to end). All seven are now `Optional`+`Computed` with
  `UseStateForUnknown`. `TestAccApplicationResource` deploys and destroys a real
  container (`nginx:alpine`) as a permanent regression test; asserts on
  `build_pack`/`limits_cpus` resolving to their known defaults.
- `GET /version` returns plain text (`4.3.2`), not JSON — the generic response decoder
  always attempted `json.Unmarshal`, so the provider's `Configure`-time connectivity
  check failed against every real Coolify instance. Found by running the acceptance
  suite against a live instance; a mocked test could not have caught it.
- `POST /projects/{uuid}/environments` only accepts `name` server-side — sending
  `description` on create was a permanent 422 despite the field being valid on update.
  `coolify_environment` now creates with `name` only and follows up with an update when
  a description is configured. Confirmed against the Laravel controller's
  `$allowedFields`, not the published OpenAPI spec, which listed it as create-time
  valid.
- `coolify_database` import used a plain UUID passthrough despite `engine` being
  `Required` with `RequiresReplace` — the attribute stayed unset after import, and the
  first plan proposed a destroy/recreate instead of a clean no-op. Import now requires
  the documented `<engine>/<uuid>` form and validates the engine against the known list.
- `coolify_notification_settings` was missing `ImportState` despite being a trivially
  importable per-channel singleton (`Read` already merges API state field-by-field
  regardless of prior values) — added, importing by channel name.
