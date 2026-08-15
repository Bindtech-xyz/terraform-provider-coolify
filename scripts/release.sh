#!/usr/bin/env bash
# Cuts a release: gate → push main to Forgejo → sync github-mirror →
# tag both → push tags. See "CI / Releases" in CLAUDE.md for the why.
#
# Usage: scripts/release.sh vX.Y.Z
#
# Idempotent-ish: safe to re-run if it fails partway, EXCEPT it refuses to
# touch a version tag that's already on a remote (never overwrites a
# published release).

set -euo pipefail

# Paths that live on main (Forgejo) but are deliberately absent from the
# github-mirror branch pushed to GitHub. Keep in sync with the "GitHub
# mirror exclusions" note in CLAUDE.md.
MIRROR_EXCLUDES=(.forgejo CLAUDE.md)

VERSION="${1:-}"
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "Usage: $0 vX.Y.Z" >&2
	exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [[ "$(git branch --show-current)" != "main" ]]; then
	echo "error: must be on main (currently on $(git branch --show-current))" >&2
	exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
	echo "error: working tree is not clean" >&2
	git status --short
	exit 1
fi

echo "==> Gate: gofmt, vet, build, test"
unformatted="$(gofmt -s -l .)"
if [[ -n "$unformatted" ]]; then
	echo "error: gofmt needed on:" >&2
	echo "$unformatted" >&2
	exit 1
fi
go vet ./...
go build ./...
go test ./... -timeout 90s

for remote_tag in "origin $VERSION" "github $VERSION"; do
	set -- $remote_tag
	remote="$1" tag="$2"
	if git ls-remote --tags "$remote" "refs/tags/$tag" | grep -q "$tag"; then
		echo "error: $tag already exists on $remote — refusing to overwrite a published release" >&2
		exit 1
	fi
done

echo "==> Pushing main to Forgejo (origin)"
git push origin main

echo "==> Syncing github-mirror"
git checkout github-mirror
exclude_pattern="$(printf '^%s(/|$)|' "${MIRROR_EXCLUDES[@]}" | sed 's/|$//')"
if ! git merge main -m "Merge main into github-mirror"; then
	conflicts="$(git diff --name-only --diff-filter=U)"
	unexpected_conflicts="$(echo "$conflicts" | grep -vE "$exclude_pattern" || true)"
	if [[ -n "$unexpected_conflicts" ]]; then
		echo "error: merge conflict outside the mirror-exclude list — resolve by hand:" >&2
		echo "$unexpected_conflicts" >&2
		git merge --abort
		git checkout main
		exit 1
	fi
	# Only excluded paths conflicted (main touched them, github-mirror has
	# them deleted): keep them deleted on this branch and continue.
	git rm -r --ignore-unmatch "${MIRROR_EXCLUDES[@]}" >/dev/null
	git commit --no-edit
fi
for path in "${MIRROR_EXCLUDES[@]}"; do
	if [[ -e "$path" ]]; then
		echo "error: $path reappeared on github-mirror after merge — should not happen" >&2
		git checkout main
		exit 1
	fi
done

echo "==> Verifying github-mirror still builds"
go build ./...

echo "==> Pushing github-mirror to GitHub main"
git push github github-mirror:main

echo "==> Tagging $VERSION"
git tag -d "$VERSION" >/dev/null 2>&1 || true
git tag -a "$VERSION" -m "$VERSION"
git push github "$VERSION"
git tag -d "$VERSION"

git checkout main
git tag -a "$VERSION" -m "$VERSION"
git push origin "$VERSION"

echo
echo "==> $VERSION pushed to both remotes. GitHub Actions release workflow:"
echo "    gh run watch --repo Bindtech-xyz/terraform-provider-coolify"
echo "    (or) https://github.com/Bindtech-xyz/terraform-provider-coolify/actions"
