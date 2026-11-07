#!/usr/bin/env bash
# Cuts a release: promotes ## Unreleased in CHANGELOG.md to a version heading,
# commits and tags it, and pushes both.
#
#   bin/justfile/release.sh major|minor|patch|X.Y.Z
#
# The tag push triggers .github/workflows/release.yml, which reads the version's
# CHANGELOG.md section back out as the release notes, so the promotion has to
# reach main before the tag does.
set -euo pipefail

bump=${1:?usage: release.sh major|minor|patch|X.Y.Z}

[[ "$(git branch --show-current)" == main ]] \
    || { echo "error: releases are cut from main" >&2; exit 1; }
git diff --quiet && git diff --cached --quiet \
    || { echo "error: working tree is dirty" >&2; exit 1; }
git fetch --quiet --tags origin main
[[ "$(git rev-parse HEAD)" == "$(git rev-parse origin/main)" ]] \
    || { echo "error: HEAD is not origin/main" >&2; exit 1; }

# A release names exactly this commit, so CI must have passed on it.
ci=$(gh run list --workflow ci.yml --commit "$(git rev-parse HEAD)" \
    --json conclusion --jq '.[0].conclusion // empty')
[[ "$ci" == success ]] \
    || { echo "error: CI hasn't passed on HEAD (got: ${ci:-no run})" >&2; exit 1; }

notes=$(awk '/^## Unreleased$/{p=1;next} p&&/^## /{exit} p' CHANGELOG.md)
[[ -n "${notes//[[:space:]]/}" ]] \
    || { echo "error: ## Unreleased is empty; nothing to ship" >&2; exit 1; }

# Bumps count from the newest stable vX.Y.Z tag, or from 0.0.0 before the
# first. Prerelease tags (v0.2.0-rc.1) don't count, so they're filtered out.
latest=$(git tag --list 'v*' | { grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' || true; } | sort -V | tail -n1)
IFS=. read -r major minor patch <<< "${latest#v}"
major=${major:-0} minor=${minor:-0} patch=${patch:-0}
case "$bump" in
    major) version="$((major + 1)).0.0" ;;
    minor) version="$major.$((minor + 1)).0" ;;
    patch) version="$major.$minor.$((patch + 1))" ;;
    *)
        [[ "$bump" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] \
            || { echo "error: want major, minor, patch, or X.Y.Z; got $bump" >&2; exit 1; }
        version=$bump ;;
esac
tag="v$version"

# Go treats v2 and up as a new module path, so a v2.0.0 tag here would build
# with a pseudo-version and break `go install`.
[[ "${version%%.*}" -lt 2 ]] \
    || { echo "error: $tag needs the module path moved to /v${version%%.*} first" >&2; exit 1; }
! git rev-parse -q --verify "refs/tags/$tag" >/dev/null \
    || { echo "error: $tag already exists" >&2; exit 1; }

# Insert the heading directly under ## Unreleased. The entries below it stay
# put, now under a version, and ## Unreleased is left empty for next time.
block=$(printf '\n## %s\n\nReleased %s.' "$version" "$(date +%Y-%m-%d)")
awk -v block="$block" '
    { print }
    /^## Unreleased$/ && !done { print block; done = 1 }
' CHANGELOG.md > CHANGELOG.md.new
mv CHANGELOG.md.new CHANGELOG.md

rm -f .git/index.lock && sleep 0.5
git add CHANGELOG.md
git commit -m "docs: cut $tag"
git tag -a "$tag" -m "$tag"

# main first: release.yml checks out the tag and fails if the version's
# changelog section isn't there.
git push origin main
git push origin "refs/tags/$tag"

echo
echo "Released $tag. Watch the workflow:"
echo "  gh run watch \$(gh run list --workflow release.yml --limit 1 --json databaseId --jq '.[0].databaseId')"
