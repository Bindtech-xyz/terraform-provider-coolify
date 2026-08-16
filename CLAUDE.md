# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Terraform provider for Coolify v4 (self-hostable PaaS), built on
`terraform-plugin-framework` v1 (protocol v6 only — no SDKv2, no muxing). Module path:
`github.com/Bindtech-xyz/terraform-provider-coolify`. Registry address baked into
`main.go`: `registry.terraform.io/bindtech-xyz/coolify`.

30 resources / 22 data sources — coverage tracks the Coolify docs sidebar (every
API-manageable concept) plus full feature parity with the community providers.
Deliberate design choices (do not "fix" these):

- `coolify_application` is ONE resource covering the five API create endpoints
  (public git / deploy key / GitHub App / dockerfile / docker image). The mode is
  derived from which source attributes are set (`applicationType()` in
  `application_resource.go`), enforced by `ConfigValidators`. `dockerfile` is
  create-only: `ApplicationsController::update_by_uuid`'s allowed-fields whitelist
  omits it entirely (`dockerfile_location`/`dockerfile_target_build` are there,
  `dockerfile` itself isn't — confirmed against the controller source; `store()`'s own
  whitelist does include it, same divergent-whitelist defect as
  `coolify_github_app.private_key_uuid` below, reported live against 4.3.2 by the same
  consumer). `applicationToRequest` only sets `Dockerfile` when `create`, and the
  schema attribute carries `RequiresReplace()` so a real content change goes through
  Create+Delete instead of a PATCH that would always 422 with `"This field is not
  allowed."` regardless of value. `dockerfile_target_build` has the exact opposite
  problem: present in `update_by_uuid`'s whitelist, absent from every create
  endpoint's — so `applicationToRequest` sends it only when `!create`, and `Create()`
  applies a configured value via a follow-up `UpdateApplication` call right after
  creating (downgraded to a warning on failure, not an error, so the resource is
  never left untracked in state over this one field — the next apply's `Read` simply
  shows it still drifted, triggering a normal `Update`). If `instant_deploy` is also
  set, Coolify's create-triggered first build already started before that follow-up
  PATCH lands, so only later deploys see the target stage. `dockerfile_location` has
  no such mismatch — sent unconditionally on both create and update. All five create
  endpoints are 3-line wrappers around one shared private `create_application()`
  method with a single `$allowedFields` (confirmed by reading past the OpenAPI
  annotations, which list fields per-endpoint for documentation only and do NOT
  reflect the actual enforcement code — a red herring that cost a round of back typing
  the wrong conclusion, `dockerfile_location` sent for a git-mode create looked
  rejectable from the annotations alone but isn't); that array and the model's
  `$fillable` both include it, and `Application::dockerfileLocation()`'s Eloquent
  mutator confirms the actual default: `/Dockerfile` only when `build_pack ===
  'dockerfile'`, `null` otherwise — conditional on build_pack, not a blanket default,
  so like `base_directory` it's `Computed`, not just `Optional`, to avoid a "provider
  produced inconsistent result" error regardless of which branch applies.
  `manual_webhook_secret_{github,gitlab,bitbucket,gitea}` (for Coolify's own
  `/webhooks/source/{provider}/events/manual` endpoints, independent of any
  git-App-managed webhook — useful when the App's own webhook URL is unreachable,
  e.g. an instance domain that must stay `http://` behind a reverse proxy) are also
  writable, unlike `coolify_github_app`'s secrets which the API can never accept back:
  confirmed against `Application::booted()`'s `creating` hook, Coolify only generates
  a random 40-character value via `??=` when the field arrives null, so a caller-supplied
  value on create is used as-is and a config-driven rotation is a normal `Update`, no
  `RequiresReplace` needed — both endpoints accept all four unconditionally (confirmed
  against the real `$allowedFields`, not the misleading per-endpoint OpenAPI
  annotations — see the `dockerfile_location` entry above for how that trap looks).
  `Optional`+`Computed`+`Sensitive`, `keepPriorIfHidden` (same `$hidden`-but-only-
  behind-`read:sensitive` treatment as `coolify_github_app.client_secret`).
- `coolify_database` is ONE resource with an `engine` attribute (8 engines) and the
  union of engine-specific fields. Coolify 422s when fields of another engine are sent;
  the client only sends non-null fields so this composes safely.
- `coolify_service.type` is intentionally NOT validated against a static enum — the
  one-click catalog (300+ templates) evolves with Coolify. `coolify_service_templates`
  fetches the live CDN feed (`client.DefaultServiceTemplatesURL`) for discovery.
- Env vars are two resources: `coolify_environment_variable`
  (application/service/database parent, addressed by uuid, updated by key) and
  `coolify_shared_environment_variable` (team/project/environment/server scope,
  addressed by numeric id).
- `coolify_storage` and `coolify_scheduled_task` follow the same parent_type/parent_uuid
  pattern as env vars. `coolify_notification_settings` (per channel) and
  `coolify_server_settings` (nested blocks per sub-endpoint) are SINGLETON resources:
  Create adopts existing settings, Delete disables (notifications) or is a no-op
  (server settings) — the API never deletes those objects. Singleton client methods use
  map[string]any bodies on purpose (Laravel fillable-driven field sets); the typed
  schema lives in the resource layer, and only configured fields are sent/refreshed so
  unmanaged settings never diff.
- `coolify_github_app` / `coolify_gitlab_app` are addressed by numeric id (not uuid) —
  the API has no single-app GET, reads filter the list. `coolify_github_app`'s
  `private_key_uuid` is genuinely unpatchable on upstream Coolify today:
  `GithubController::update()` validates it with Laravel's `uuid` rule (RFC 4122)
  before looking it up with `PrivateKey::where('uuid', ...)`, but Coolify's own
  private key ids are short nanoids — no value can satisfy both, confirmed by
  reading the controller source directly (`store()`, the POST path, uses the
  correct `string` rule on the same field one call earlier — an upstream
  regression, not a real constraint). `githubAppUpdateRequest` omits the field
  from the PATCH body whenever it's unchanged from state so an unrelated apply
  never 422s on it. Read/Import always resolve `private_key_id` (echoed on every
  response — `GithubApp.$hidden` only covers the two secrets) to its uuid via
  `resolvePrivateKeyUUID` (a `/security/keys` list scan) instead of trusting
  whatever was last configured, which is what used to leave the field
  permanently null after import and trigger an update — hence this 422 — on
  every single apply. Changing the private key for real still 422s until
  upstream fixes the validator. `client_id`/`client_secret`/`webhook_secret` are
  `Optional`+`Computed` (not `Required`): confirmed live that Coolify echoes
  `client_secret`/`webhook_secret` in full to any token with `read:sensitive`
  (`GithubApp.$hidden` is overridden per-request by `removeSensitiveData()`'s
  `makeVisible()`) and `client_id` unconditionally (never sensitive), so
  `keepPriorIfHidden` adopts the real values on import instead of forcing
  required-but-unobtainable config values — a plain `terraform import` is fully
  clean, nothing to stash in a vault first. A token without `read:sensitive` still
  falls back to whatever was
  configured, same as every other sensitive field.
- Deletes of applications/databases/services/destinations call
  `client.waitForDeletion` (poll until 404, backoff 500ms→5s) because Coolify tears
  down asynchronously; without it, destroy-then-recreate collides on held
  names/domains/networks. Client unit tests that exercise those deletes MUST answer
  404 on the post-delete GET or they hang for the poll deadline.
- `coolify_cloud_server` is ONE resource for the 3 VM providers (provider_name
  discriminator, `cloudServerBody` resolves the image-field type difference:
  Hetzner int id vs DigitalOcean string slug). Destroy only deregisters — the VM
  survives at the provider (warning emitted).
- `coolify_volume_backup` has NO read API (PUT upsert only): Read echoes state,
  drift reconciles on apply. `coolify_resource_action` is fire-and-forget
  (re-runs via `triggers` map, RequiresReplace everywhere).
- `coolify_environment_variables` (bulk map) vs `coolify_environment_variable`
  (unitary, per-flag): both valid, keys must not overlap on one parent.
- `coolify_application_preview` has **no create API call at all** — Coolify creates
  previews only from GitHub App PR webhook events, never directly. `Create` is
  state-only (tracks `application_uuid`+`pull_request_id`); `Delete` is the only real
  API interaction. Use it to guarantee cleanup, not to provision the preview.
- `coolify_api_settings` (instance-wide REST API + MCP server toggles, root-team token
  only) has no GET for either flag — same no-read-API constraint as
  `coolify_volume_backup`/`coolify_server_settings`. `Delete` **unconditionally**
  re-enables the API and disables MCP regardless of the resource's last configured
  state — there is no other way back in once the API is off, so this refuses to leave
  that footgun loaded. Never test `api_enabled = false` against a real instance you
  still need API access to; the acceptance suite must not exercise it either.
  `allowed_ips` (the instance IP allowlist — a separate gate from both Cloudflare
  Access and the bearer token, see the live-testing protocol note in this file's
  history) is **deliberately not on this resource, and can't be**: it has no REST API
  route at all, only the Livewire settings UI (`app/Livewire/Settings/Advanced.php`,
  confirmed by grepping `routes/api.php`). Don't add it here; there is no endpoint to
  call. Reported by a consumer project running this instance behind a Cloudflare
  Tunnel: Coolify seeds `allowed_ips` with its own `public_ipv4` at install time, and
  since tunnel traffic never arrives from that address, the 403 ("You are not allowed
  to access the API.") can reappear on its own even after being fixed once — it isn't
  necessarily a regression, check the current value before assuming something broke.
- `coolify_resource_tag` attaches a tag (creating it if needed) to an
  application/database/service. `tag_name` is Required but never adopted from the API
  into state — Coolify lowercases tag names server-side, so state always echoes config
  instead (same reasoning as `coolify_cloud_init_script.script`).

## Commands

```sh
make build                        # go build ./...
make test                         # unit tests (client tests use httptest, no credentials)
make lint                         # golangci-lint run
make docs                         # go generate → tfplugindocs regenerates docs/
make testacc                      # ALL acceptance tests — hits a real Coolify instance
make release VERSION=v0.1.1       # gate, sync github-mirror, tag + push both remotes

# Single acceptance test:
COOLIFY_ENDPOINT=... COOLIFY_TOKEN=... TF_ACC=1 \
  go test ./internal/provider/ -v -run TestAccProjectResource

# Single unit test:
go test ./internal/client/ -run TestCreateProjectFollowsUpWithRead -v
```

Acceptance tests create/destroy real objects; they need `COOLIFY_ENDPOINT`,
`COOLIFY_TOKEN` and `TF_ACC=1`, and should only target a disposable instance.

Local manual testing uses `dev_overrides` in `~/.terraformrc` pointing at GOBIN +
`make install`; with dev_overrides active, do NOT run `terraform init` (see README).

## Architecture

Two layers, strictly separated:

- `internal/client/` — hand-written HTTP client for the Coolify REST API. No framework
  imports here. One file per API object (`project.go`, `server.go`) + shared transport
  (`client.go`) and error taxonomy (`errors.go`). All endpoints live under `/api/v1`,
  bearer-token auth. `client.New` normalises the endpoint (appends `/api/v1`, tolerates
  it being present). `do()` retries 429 (honouring Retry-After) and 5xx only.
- `internal/provider/` — framework layer. One file per resource
  (`<name>_resource.go`) / data source (`<name>_data_source.go`) plus its `_test.go`.
  `provider.go` owns configuration (endpoint/token/insecure, env fallbacks
  `COOLIFY_ENDPOINT`/`COOLIFY_TOKEN`) and registers constructors; `helpers.go` has the
  `types.X` → `*T` converters and Configure type-assertion helpers.

Patterns the existing code relies on — keep them when adding objects:

- **Create returns only `{"uuid"}`** on Coolify create endpoints. Client `CreateX`
  methods therefore POST then immediately GET to return a full object. Same for
  `UpdateX` (PATCH then GET).
- **Request structs use pointer fields** with `omitempty` so unset Terraform attributes
  are omitted from JSON rather than sent as zero values (Coolify would apply them).
- **Read must detect drift**: on `client.IsNotFound(err)` call
  `resp.State.RemoveResource(ctx)` and return, don't error.
- **Null-preserving mapping**: API normalises absent strings to `""`; `xToModel`
  functions keep the attribute null if it was null in prior state/config (helpers
  `keepNullIfEmpty` / `keepPriorIfHidden` in `helpers.go`) to avoid permanent diffs.
  Write-only attributes the API never echoes (e.g. `coolify_cloud_init_script.script`)
  are carried over from prior state. Sensitive fields (db credentials, env-var values,
  S3 keys, `coolify_github_app.client_secret`/`webhook_secret`) are only echoed to
  tokens with the `read:sensitive` ability — `keepPriorIfHidden` covers both cases.
  Don't assume a field is write-only just because a Laravel model's `$hidden` lists it:
  `$hidden` is commonly overridden per-request (`makeVisible()` gated on
  `read:sensitive`, as `GithubController::removeSensitiveData()` does) — confirm
  against a real response, not the model source alone, before treating a field as
  never-echoed (found live: `coolify_github_app.client_secret`/`webhook_secret` were
  wrongly assumed write-only and hardcoded `Required`, until a `read:sensitive` token
  showed they come back in full).
- Computed-once attributes (`uuid`) get `stringplanmodifier.UseStateForUnknown()`.
- Every resource implements `ImportState` (passthrough on `uuid`, not `id`).
- Interface-conformance `var _ resource.Resource = ...` guards top each file.

Adding a new object = client file + resource/data-source file + registration in
`provider.go` `Resources()`/`DataSources()` + example under `examples/` + acceptance
test + `make docs`.

## API reference

The REAL source of truth is the Laravel controllers in the Coolify repo
(`app/Http/Controllers/Api/*.php` + `routes/api.php` on
https://github.com/coollabsio/coolify) — each handler declares an `$allowedFields`
array and validator rules; extra fields are rejected with 422 "This field is not
allowed". The published OpenAPI (`openapi.json` in that repo) lags the controllers
(e.g. it shows `PATCH /security/keys` without `{uuid}` while routes have it). When
adding endpoints, grep the controller, not the spec. Error bodies are Laravel-style:
`{"message": ...}` plus `{"errors": {field: [msgs]}}` on 422; 429 carries Retry-After
(honoured by the client's retry loop). Most create endpoints return only
`{"uuid": ...}` (shared env vars return `{"id": ...}`; destinations return the full
object). `POST /deploy` is a rare case where the *same endpoint* returns two different
envelopes depending on which mutually-exclusive query param was used —
`DeployController::by_uuids` wraps results in `{"deployments": [...]}` (per-entry
`message`), `::by_tags` in `{"details": [...], "message": [...]}` (aggregate message,
no per-entry one). `client.Deploy` decodes both into the same `[]DeployResult`; found
live when the by-tag path silently returned zero results (no error) because only
`"deployments"` was decoded. Tag names are also always lowercased server-side on
attach (`coolify_resource_tag`) but matched *exactly* (case-sensitive) on deploy — both
`findTag` and `deploymentResource.trigger` normalize to lowercase client-side to
compensate; don't remove either normalization.

Watch for divergent `$allowedFields` whitelists between an object's create and update
controller methods — found three times so far, all reported live by the same consumer
against 4.3.2, all real upstream regressions rather than intentional immutability:
`GithubController::update()` validates `private_key_uuid` with Laravel's `uuid` rule
(RFC 4122) against Coolify's own nanoid ids; `ApplicationsController::update_by_uuid`
drops `dockerfile` from its whitelist outright while `store()` requires it on create;
and the same method's whitelist *adds* `dockerfile_target_build`, which no create
endpoint accepts at all — the mismatch runs both directions, not just
create-only-missing-from-update. None have an upstream issue filed yet as of this
writing. The provider-side mitigation shape depends on which direction the field is
missing from: never send it on `PATCH` at all, either by comparing against state (the
field is legitimately still updatable in principle, e.g. `private_key_uuid`) or with
`RequiresReplace()` (genuinely create-only, e.g. `dockerfile`) — or never send it on
`POST`, applying a configured value via a follow-up `PATCH` right after create instead
(update-only, e.g. `dockerfile_target_build`). Don't assume any other object's
create/update field lists actually match without checking both controller methods
directly.

## CI / Releases

The repo lives on Forgejo (`Applications/terraform-provider-coolify` on
git.lan.bdigitalservices.com, remote `origin`); `.forgejo/workflows/` is the CI that
actually runs there. It is published to `registry.terraform.io` from a GitHub mirror
(remote `github`, `github.com/Bindtech-xyz/terraform-provider-coolify` — registry
ingestion only works from GitHub). Keep `.forgejo/workflows/` and `.github/workflows/`
in sync when editing CI logic, but **do not push `main` straight to `github`** — the
`github-mirror` branch is what actually gets pushed there, and it deliberately omits a
few paths (see **GitHub mirror exclusions** below).

`secrets/` is gitignored (maintainer's personal acceptance-test tooling, depends on a
machine-local SOPS/age config outside this repo) — exists locally, never tracked.
`Taskfile.yml` is different: it no longer wraps `secrets/`, just `scripts/release.sh`,
so it's plain project tooling and *is* tracked (both remotes).

### Cutting a release

`task bugs:ship MSG="..." VERSION=vX.Y.Z` (fix the bug yourself first) — commits on
`main`, then runs `scripts/release.sh vX.Y.Z`: gate (gofmt/vet/build/test) → push
`main` to `origin` → sync `github-mirror` → push it to `github`'s `main` → tag both
remotes → push both tags. Refuses to run from a dirty tree, off `main`, or over a tag
that already exists on either remote. Full runbook, troubleshooting, and one-time
setup notes: **`RELEASING.md`** (also excluded from `github-mirror` — see below).

Tag `vX.Y.Z` → GitHub's release workflow runs GoReleaser (config in `.goreleaser.yml`):
signs SHA256SUMS with the GPG key from repo secrets (`GPG_PRIVATE_KEY`, `PASSPHRASE`)
and attaches `terraform-registry-manifest.json` (protocol 6.0). Forgejo's own release
workflow does the same against `GITEA_TOKEN` if that remote's tag is pushed too.
Registry repo naming/tag conventions are load-bearing: repo must stay
`terraform-provider-coolify`, tags semver with `v`. Once a version's GitHub release
exists, `registry.terraform.io` picks it up automatically (observed delay: ~15–30s) —
no manual step after the first `Publish provider` link-up.

### GitHub mirror exclusions

`github-mirror` is a merge-forward branch (never rebased/rewritten, so the push to
`github` is always a plain fast-forward — no force-push needed) that tracks `main`
minus `MIRROR_EXCLUDES` in `scripts/release.sh`, currently:

- `.forgejo/` — meaningless on GitHub (GitHub Actions can't run Forgejo Actions
  workflows), and no secrets in it, just irrelevant CI config.
- `CLAUDE.md` — internal maintainer/AI-agent guidance (this file), not something a
  public OSS consumer needs.
- `RELEASING.md` — internal release runbook; same reasoning.

To add another exclusion, add it to `MIRROR_EXCLUDES` in `scripts/release.sh` — the
merge-conflict auto-resolution there already generalizes over the whole list. Manual
sync (e.g. mid-development, not at release time) works the same way the script does it
internally:

```sh
git checkout github-mirror
git merge main
# if main touched an excluded path, it comes back here too — drop it again:
git rm -r --ignore-unmatch .forgejo CLAUDE.md && git commit --no-edit
git push github github-mirror:main
git checkout main
```

## Docs

`docs/` is generated by tfplugindocs from schema `MarkdownDescription`s + `examples/`
+ `templates/` — edit those sources, never `docs/` directly.
