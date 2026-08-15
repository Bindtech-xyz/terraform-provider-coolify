# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versioning follows
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.2.0] - 2026-08-15

### Added

- Five resources closing every real gap found comparing against
  `coolify-terraform` (56 resources total there vs 25 here at the time — most of that
  gap was deliberate consolidation already documented in this provider's design
  choices, e.g. one `coolify_application`/`coolify_database` instead of one resource
  per mode/engine, but five capabilities were genuinely missing, not just modeled
  differently):
  - `coolify_deployment` — triggers a deployment by `resource_uuid` or `tag`
    (`POST /deploy`), with optional `wait_for_completion` that polls until the
    deployment reaches `finished`/`failed`. The client already had a fully-built,
    tested `Deploy` method with nothing calling it — dead code until now.
  - `coolify_resource_tag` — attaches a tag to an application/database/service (the
    existing `coolify_tag` only creates team-wide tags, never attaches them).
  - `coolify_application_destination` — attaches an additional standalone Docker
    destination to an application for multi-destination deployment.
  - `coolify_application_preview` — tracks a PR preview deployment for cleanup on
    destroy (Coolify has no create/read API for previews; they're created only from
    GitHub App PR webhooks).
  - `coolify_api_settings` — instance-wide REST API and MCP server enable/disable
    (root-team token only).

### Fixed

- `coolify_deployment` crashed on every real create with "Value Conversion Error":
  `results` (Computed-only `ListNestedAttribute`) is unknown in the plan on create
  (no prior state to fall back to), and the framework cannot decode an unknown value
  into a plain Go slice field. Fixed by typing `Results` as `types.List` (which can
  represent unknown) instead of `[]deployResultModel`, and adding
  `UseStateForUnknown` so a later `Update` (only reachable via
  `wait_for_completion`/`timeout_seconds`, since every other attribute forces
  replacement) doesn't hit the same problem.
- `client.Deploy` silently returned zero results for every tag-based deploy — no
  error, just nothing. `POST /deploy` returns genuinely different response envelopes
  depending on which mutually-exclusive query param triggered it:
  `DeployController::by_uuids` wraps results in `{"deployments": [...]}`, `::by_tags`
  in `{"details": [...], "message": [...]}` instead. The client only decoded
  `"deployments"`. Found live deploying `tf-sweep4-web` by tag and getting an empty
  `results` back with no diagnostic at all.
- Deploying by tag also 404'd outright before the envelope fix even mattered:
  `POST /deploy?tag=...` matches the tag name *exactly* (case-sensitive) against
  Coolify's stored value, which is always lowercased on attach
  (`Tag::createOrFirst`/`normalizeTagNames` server-side). A `coolify_deployment.tag`
  configured with any uppercase character (matching how the tag was originally typed
  in `coolify_resource_tag.tag_name`) silently matched nothing. Both
  `coolify_resource_tag`'s `findTag` (comparing) and `coolify_deployment`'s `trigger`
  (before sending to `/deploy`) now normalize to lowercase client-side.

Verified live: `coolify_resource_tag` (mixed-case tag name, confirming the
normalization fix), `coolify_deployment` by uuid with `wait_for_completion = true`
(a real deploy, polled to `finished`), `coolify_deployment` by tag (after both fixes
above), `coolify_application_preview` (state-only create, clean destroy against a
nonexistent preview), and `coolify_api_settings` (`mcp_enabled` toggle only —
`api_enabled = false` was deliberately never tested against the real instance, since
there would be no way back in except the UI).
`coolify_application_destination` could not be verified live — the test instance has
only one server, and attaching a second destination requires one on a different
server than the primary.

## [0.1.2] - 2026-08-15

### Added

- Documented (README and the registry overview page) how to use this provider under
  OpenTofu today: it isn't listed on `registry.opentofu.org` yet, so `tofu init`
  resolves an unqualified `source` against the wrong registry and fails. Qualifying
  the hostname explicitly (`source = "registry.terraform.io/bindtech-xyz/coolify"`)
  works right now — verified directly against a real release: signature
  verification and install both succeed under `tofu init`.

## [0.1.1] - 2026-08-15

### Changed

- Moved everything that had accumulated under `[Unreleased]` to a dated `[0.1.0]`
  section, matching Keep a Changelog convention — the initial release had already
  shipped, so leaving it all under `[Unreleased]` was stale.

## [0.1.0] - 2026-08-15

### Added

- Provider-level `headers` attribute: a generic, sensitive map of fixed HTTP headers
  sent with every request, applied before (and unable to override) the provider's own
  `Authorization` header. Not modeled after any specific reverse proxy — it exists for
  reaching a Coolify instance behind any authenticating edge layer (Cloudflare Access
  service tokens, `oauth2-proxy`, a header-based mTLS gateway, ...). Unknown values
  (e.g. an edge-auth resource's attributes not yet applied) produce an explicit
  "provider block cannot depend on a same-apply resource" diagnostic instead of an
  opaque connectivity failure. See the
  [Reaching Coolify Behind an Authenticating Edge](https://registry.terraform.io/providers/bindtech-xyz/coolify/latest/docs/guides/reverse-proxy-authentication)
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
  [Token Abilities & Sensitive Data](https://registry.terraform.io/providers/bindtech-xyz/coolify/latest/docs/guides/authentication-and-token-abilities)
  guide.
- Composite import support for every resource whose identity spans more than one
  coordinate (`coolify_environment`, `coolify_storage`, `coolify_scheduled_task`,
  `coolify_database_backup`, `coolify_volume_backup`, `coolify_shared_environment_variable`,
  and `coolify_database`'s `<engine>/<uuid>` form) — documented end to end in the
  [Importing Existing Infrastructure](https://registry.terraform.io/providers/bindtech-xyz/coolify/latest/docs/guides/importing-existing-infrastructure)
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
- Nine list data sources (`coolify_tags`, `coolify_destinations`, `coolify_s3_storages`,
  `coolify_teams`, `coolify_backup_executions`, `coolify_github_app_repositories`,
  `coolify_cloud_catalog`, `coolify_server_domains` — both its outer list and each row's
  nested `domains`, `coolify_server_resources`) initialized their list-typed model field
  as a bare Go zero-value struct instead of `make([]T, 0, len(source))`. A Go nil slice
  and an empty slice both marshal to an empty Terraform list only if the field was ever
  initialized as a slice at all — the zero value of a slice field left untouched is
  `nil`, which the framework instead marshals as Terraform `null`. Any API response with
  zero items (a fresh instance, a team with no tags yet, ...) then produced `null`
  instead of `[]`, and Terraform Core's `length()` — along with any real `for_each` or
  `toset()` on the output — rejects `null` outright with "argument must not be null."
  Found via a live comprehensive-sweep apply computing `length()` over every list data
  source's output; confirmed fixed by re-running the same apply against a real instance
  with zero tags/S3 storages at the time. `TestDataSourcesReturnEmptyListNotNull` locks
  in the empty-array case for three of the nine against a real `httptest` server.
- `coolify_database_backup.databases_to_backup`: the same "Optional-only, but Coolify
  assigns a non-empty default" class as several fixes above — unset resolves to the
  engine's own logical database name (e.g. `postgres`), never `""`. Found deploying a
  real `postgresql` backup schedule end to end. Now `Optional`+`Computed` with
  `UseStateForUnknown`.
- `coolify_cloud_init_script.script`: Coolify strips a trailing newline from stored
  script content server-side. `script` is `Required` (Required+Computed is not a valid
  attribute combination), so the planned value is always known — any byte-for-byte
  difference from what Create/Update returns is a hard "provider produced inconsistent
  result" error, on essentially every script written with a trailing newline (i.e.
  nearly all of them — that's how every editor and HCL heredoc writes files). Found
  deploying a real cloud-init script end to end. Fixed by echoing the configured value
  back into state on create/read/update instead of adopting the API's normalized value;
  import (no prior config to echo) still adopts the API's value.
- `coolify_server.instant_validate`'s description claimed a bad key/IP would "fail the
  apply instead of leaving a broken server behind." That's not how Coolify behaves:
  `create-server` dispatches SSH validation as an asynchronous queued job and always
  returns 201 immediately, regardless of `instant_validate` — `is_reachable`/`is_usable`
  read back `false` right after `apply` even with a correct key and IP, and only reflect
  reality once the background check completes. Found live: a server created against an
  unreachable RFC 5737 test address succeeded outright instead of failing the apply as
  documented. Corrected the description; no code change was needed since the
  Computed/Default plumbing was already right, only the claim about it was wrong.
- Documented, on every resource with a free-form `description` (`application`,
  `database`, `environment`, `project`, `server`, `s3_storage`, `service`), the
  character restriction Coolify enforces server-side (letters, numbers, whitespace, and
  `- _ . , ! ? ( ) ' " + = * @ / &` — notably no colon or semicolon) that previously
  surfaced only as an opaque 422 on first use. Found live creating a project whose
  description used a colon. `coolify_private_key.description` is intentionally
  unchanged — that field validates as plain `string|max:255` with no such restriction.

### Verified

- Every one of the 25 resources and 22 data sources has now been exercised against a
  real Coolify instance at least once — including `coolify_server`, `coolify_tag`,
  `coolify_destination`, `coolify_shared_environment_variable`, `coolify_s3_storage`,
  `coolify_scheduled_task`, `coolify_notification_settings`, and the singular/plural
  data source pairs (`coolify_project`/`coolify_projects`, `coolify_server`/
  `coolify_servers`, `coolify_team`/`coolify_teams`) and `coolify_instance`, which a
  first sweep had missed. Where a resource depends on a genuine external system (a real
  cloud provider account, a real SMTP/S3 endpoint), it was exercised with well-formed
  but non-functional credentials, confirming the full request chain reaches Coolify's
  real server-side validation and that validation surfaces as a clean diagnostic rather
  than a provider crash — the same bar every earlier fix in this changelog was held to.
