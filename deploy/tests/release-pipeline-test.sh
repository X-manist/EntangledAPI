#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CUSTOM_WORKFLOW="$ROOT_DIR/.github/workflows/custom-release-on-merge.yml"
RELEASE_WORKFLOW="$ROOT_DIR/.github/workflows/release.yml"
SIMPLE_CONFIG="$ROOT_DIR/.goreleaser.simple.yaml"

fail() {
    printf 'FAIL: %s\n' "$*" >&2
    exit 1
}

grep -Fq 'group: release-publish' "$RELEASE_WORKFLOW" \
    || fail 'Release publishing does not use the global concurrency group'
grep -Fq 'queue: max' "$RELEASE_WORKFLOW" \
    || fail 'Release publishing does not preserve queued releases'
grep -Fq 'run-name: Release ${{ inputs.tag }}' "$RELEASE_WORKFLOW" \
    || fail 'Release run name is not tied to the requested tag'
grep -Fq 'SIMPLE_RELEASE: ${{ inputs.simple_release }}' "$RELEASE_WORKFLOW" \
    || fail 'Automatic Release can still be forced into simple mode by a repository variable'
grep -Fq 'make_latest: false' "$SIMPLE_CONFIG" \
    || fail 'Simple releases can still replace the complete latest release'
if grep -Fq '/sub2api:latest' "$SIMPLE_CONFIG"; then
    fail 'Simple releases can still replace the GHCR latest image'
fi

timestamp_line=$(grep -nF 'dispatched_at=$(date' "$CUSTOM_WORKFLOW" | head -1 | cut -d: -f1)
dispatch_line=$(grep -nF 'status=$(curl' "$CUSTOM_WORKFLOW" | head -1 | cut -d: -f1)
[ -n "$timestamp_line" ] && [ -n "$dispatch_line" ] && [ "$timestamp_line" -lt "$dispatch_line" ] \
    || fail 'Dispatch lower-bound timestamp is not captured before the API request'
grep -Fq '.display_title == $title' "$CUSTOM_WORKFLOW" \
    || fail 'Release waiter does not match the exact tag-specific run name'

grep -Fq 'archive_name="sub2api_${release_version}_linux_amd64.tar.gz"' "$RELEASE_WORKFLOW" \
    || fail 'Existing Release validation does not require the exact tag archive'
grep -Fq '.name == $archive and ((.size // 0) > 0)' "$RELEASE_WORKFLOW" \
    || fail 'Existing Release validation does not require a non-empty exact archive'
grep -Fq 'Existing checksums.txt has no valid SHA-256 entry' "$RELEASE_WORKFLOW" \
    || fail 'Existing Release validation does not inspect the checksum entry'
grep -Fq 'git tag --merged "origin/$CUSTOM_RELEASE_BRANCH"' "$RELEASE_WORKFLOW" \
    || fail 'Manual dispatch can publish a tag outside the production tag order'
grep -Fq 'RELEASE_TAG" != "$latest_production_tag' "$RELEASE_WORKFLOW" \
    || fail 'Historical production tags are not rejected before publication'

if command -v ruby >/dev/null 2>&1; then
    ruby -e 'require "yaml"; ARGV.each { |path| YAML.load_file(path) }' \
        "$CUSTOM_WORKFLOW" "$RELEASE_WORKFLOW" "$SIMPLE_CONFIG"
fi

printf 'release pipeline static checks passed\n'
