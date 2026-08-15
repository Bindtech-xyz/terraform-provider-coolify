# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Terraform provider for Coolify v4 (self-hostable PaaS), built on
`terraform-plugin-framework` v1 (protocol v6 only — no SDKv2, no muxing). Module path:
`github.com/Bindtech-xyz/terraform-provider-coolify`. Registry address baked into
`main.go`: `registry.terraform.io/bindtech-xyz/coolify`.

25 resources / 22 data sources — coverage tracks the Coolify docs sidebar (every
API-manageable concept) plus full feature parity with the community providers.
Deliberate design choices (do not "fix" these):

- `coolify_application` is ONE resource covering the five API create endpoints
  (public git / deploy key / GitHub App / dockerfile / docker image). The mode is
  derived from which source attributes are set (`applicationType()` in
  `application_resource.go`), enforced by `ConfigValidators`.
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
  the API has no single-app GET, reads filter the list.
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

## Commands

```sh
make build                        # go build ./...
make test                         # unit tests (client tests use httptest, no credentials)
make lint                         # golangci-lint run
make docs                         # go generate → tfplugindocs regenerates docs/
make testacc                      # ALL acceptance tests — hits a real Coolify instance

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
  Write-only attributes the API never echoes (e.g. `private_key_uuid`) are carried over
  from prior state. Sensitive fields (db credentials, env-var values, S3 keys) are only
  echoed to tokens with the `read:sensitive` ability — `keepPriorIfHidden` covers both
  cases.
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
object).

## CI / Releases

The repo lives on Forgejo (`Applications/terraform-provider-coolify` on
git.lan.bdigitalservices.com, remote `origin`); `.forgejo/workflows/` is the CI that
actually runs there. It is published to `registry.terraform.io` from a GitHub mirror
(remote `github`, `github.com/Bindtech-xyz/terraform-provider-coolify` — registry
ingestion only works from GitHub). Keep `.forgejo/workflows/` and `.github/workflows/`
in sync when editing CI logic, but **do not push `main` straight to `github`** —
`.forgejo/workflows/` has no purpose there and was deliberately dropped from that
mirror. Instead:

```sh
git checkout github-mirror
git merge main              # bring in new commits from main
# if main touched .forgejo/workflows/, it comes back here too — remove it again:
git rm -r .forgejo/ 2>/dev/null && git commit -m "drop .forgejo/workflows again"
git push github github-mirror:main
git checkout main
```

This is a merge-forward branch, never rebased/rewritten, so the push is always a plain
fast-forward — no force-push needed. `secrets/` and `Taskfile.yml` are gitignored
(maintainer's personal acceptance-test tooling, depends on a machine-local SOPS/age
config outside this repo) — they exist locally but were deliberately never meant to be
tracked going forward.

Tag `vX.Y.Z` → release workflow runs GoReleaser (config in `.goreleaser.yml`): signs
SHA256SUMS with the GPG key from repo secrets and attaches
`terraform-registry-manifest.json` (protocol 6.0). GoReleaser picks its target from the
token env var: `GITEA_TOKEN` → Forgejo release (URLs in `gitea_urls`), `GITHUB_TOKEN` →
GitHub release. Registry repo naming/tag conventions are load-bearing: repo must stay
`terraform-provider-coolify`, tags semver with `v`.

## Docs

`docs/` is generated by tfplugindocs from schema `MarkdownDescription`s + `examples/`
+ `templates/` — edit those sources, never `docs/` directly.
