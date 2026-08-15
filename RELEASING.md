# Releasing

Internal runbook for cutting a release. Not needed to build, test, or use the
provider — see `CLAUDE.md` for why this file (like `.forgejo/`) is excluded from
the `github-mirror` branch pushed to GitHub.

## TL;DR

```sh
task bugs:ship MSG="fix: describe the bug" VERSION=v0.1.2
```

Fix the bug yourself first (edit the files). Once that's done and you're ready to
ship, the one command above does everything else. Nothing further to remember.

## What it actually does

`task bugs:ship` (`Taskfile.yml`) is a thin wrapper: it commits, then hands off to
`scripts/release.sh`, which does the rest. In order:

1. **Commit on `main`** — `git add -A && git commit -m "$MSG"`. Requires `MSG` and
   `VERSION`, and requires you to already be on `main` with no other uncommitted
   surprises (the precondition checks this and refuses otherwise).
2. **Gate** — `gofmt -s -l .`, `go vet ./...`, `go build ./...`, `go test ./...`.
   Aborts before touching any remote if any of these fail.
3. **Push `main` to Forgejo** (`origin`) — the canonical repo, where
   `.forgejo/workflows/` actually runs CI.
4. **Sync `github-mirror`** — checks out the `github-mirror` branch, merges `main`
   into it, and (if `main` touched an excluded path since the last sync) re-removes
   `MIRROR_EXCLUDES` from `scripts/release.sh` — currently `.forgejo/` and
   `CLAUDE.md`. This branch is only ever merged forward, never rebased, so the next
   step is always a plain fast-forward push, no force required.
5. **Push `github-mirror` to GitHub's `main`** (remote `github`).
6. **Tag + push both remotes** — `vX.Y.Z` is created and pushed to `github` first
   (on the `github-mirror` commit), then to `origin` (on the `main` commit).
   Refuses outright if the tag already exists on either remote — it will never
   overwrite a published release.

## What happens after the push

Pushing the tag to `github` triggers `.github/workflows/release.yml`:

1. Imports `GPG_PRIVATE_KEY` (repo secret) with `PASSPHRASE` (empty — the key has
   none, generated non-interactively for headless CI signing).
2. Runs GoReleaser (`.goreleaser.yml`): cross-compiles for every OS/arch pair
   Terraform's registry expects, zips each, computes `SHA256SUMS`, signs it
   (`SHA256SUMS.sig`), attaches `terraform-registry-manifest.json`, and publishes
   everything as a GitHub Release.
3. `registry.terraform.io` picks up the new version automatically via the webhook
   set up when the provider was first linked (Publish → Provider, one-time, already
   done for `Bindtech-xyz/coolify`) — no manual step. Observed propagation delay:
   roughly 15–30 seconds after the GitHub Release finishes publishing before
   `GET /v1/providers/bindtech-xyz/coolify/versions` lists it.

Pushing the tag to `origin` (Forgejo) triggers `.forgejo/workflows/release.yml` the
same way, against `GITEA_TOKEN` instead — a Forgejo-side release, independent of the
GitHub/registry path above.

## If it fails partway

`scripts/release.sh` is safe to re-run: it refuses to redo anything already
confirmed done (existing tags are never overwritten), and every step up to tagging
is naturally idempotent (re-pushing an unchanged branch/commit is a no-op).

- **Gate fails**: fix the code, `git commit --amend` or a new commit, re-run
  `task bugs:ship` with the same `VERSION`.
- **Merge conflict on `github-mirror` outside `MIRROR_EXCLUDES`**: the script aborts
  the merge and returns you to `main` — resolve by hand (see "Manual sync" in
  `CLAUDE.md`), then re-run.
- **CI (Lint/Tests) goes red on GitHub after a push**: doesn't block the release
  workflow (separate trigger — tag push vs. branch push), but fix it in a follow-up
  commit regardless. `golangci-lint-action` needs to stay on a major version that
  supports whatever golangci-lint version is pinned in `.github/workflows/test.yml`
  and `.forgejo/workflows/test.yml` — this broke once already (v6 silently resolved
  an incompatible v1.x build, then hard-errored on v2 entirely; both are pinned
  explicitly now, see the "Fix CI" and "Fix Lint" commits from 2026-08-15 for the
  full story).
- **Release workflow fails on GitHub** (e.g. bad GPG import): the tag is already
  pushed, so re-running means either deleting the tag on `github` first
  (`git push github :refs/tags/vX.Y.Z`) and re-running the script, or just
  re-triggering the existing workflow run (`gh run rerun <id>`) once the underlying
  issue (usually a secret) is fixed.

## One-time setup (already done, kept for disaster recovery)

- GitHub org `Bindtech-xyz`, repo `terraform-provider-coolify`, SSH remote `github`.
- GPG key (RSA 4096, no passphrase — required for unattended CI signing): public
  half registered on the maintainer's registry.terraform.io account (Settings →
  Signing Keys); private half + empty `PASSPHRASE` set as GitHub Actions repo
  secrets (`GPG_PRIVATE_KEY`, `PASSPHRASE`). Encrypted backup of the private key
  lives in `secrets/terraform.enc.yaml` (`gpg_private_key`, `gpg_fingerprint`) —
  gitignored, machine-local, see `CLAUDE.md`.
- Provider linked on registry.terraform.io: Publish → Provider →
  `Bindtech-xyz/terraform-provider-coolify`. This is what makes future tags
  auto-publish; it only needs doing once per provider, not per release.
