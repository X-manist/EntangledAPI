#!/usr/bin/env bash

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${TEST_DIR}/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-install-test.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

fail() {
    printf 'FAIL: %s\n' "$*" >&2
    exit 1
}

assert_eq() {
    local expected="$1"
    local actual="$2"
    local label="$3"
    [ "$expected" = "$actual" ] || fail "$label: expected '$expected', got '$actual'"
}

export SUB2API_INSTALL_LIBRARY_ONLY=true
# shellcheck source=../install.sh
source "${DEPLOY_DIR}/install.sh"

RELEASE_CHANNEL=custom
REPOSITORY_OVERRIDE=
configure_release_source
assert_eq "X-manist/EntangledAPI" "$GITHUB_REPO" "custom channel"

RELEASE_CHANNEL=official
REPOSITORY_OVERRIDE=
configure_release_source
assert_eq "Wei-Shaw/sub2api" "$GITHUB_REPO" "official channel"

RELEASE_CHANNEL=custom
REPOSITORY_OVERRIDE=example/private-release
configure_release_source
assert_eq "example/private-release" "$GITHUB_REPO" "repository override"

if (RELEASE_CHANNEL=invalid; REPOSITORY_OVERRIDE=; configure_release_source >/dev/null 2>&1); then
    fail "invalid channel was accepted"
fi
if (RELEASE_CHANNEL=custom; REPOSITORY_OVERRIDE=https://github.com/example/repo; configure_release_source >/dev/null 2>&1); then
    fail "invalid repository was accepted"
fi

is_interactive() {
    return 1
}
help_output=$(main --help --channel official --repository example/override 2>&1)
printf '%s' "$help_output" | grep -q 'Release source: official (example/override)' \
    || fail "CLI options did not override the release source"

release_json='{
  "tag_name": "v1.2.3-entangled.4",
  "assets": [
    {"name":"sub2api_1.2.3-entangled.4_linux_amd64.tar.gz","url":"https://api.github.com/repos/example/private-release/releases/assets/10"},
    {"name":"checksums.txt","url":"https://api.github.com/repos/example/private-release/releases/assets/11"}
  ]
}'
asset_url=$(release_asset_api_url "$release_json" "checksums.txt")
assert_eq "https://api.github.com/repos/example/private-release/releases/assets/11" "$asset_url" "asset lookup"
if release_asset_api_url "$release_json" "missing.tar.gz" >/dev/null 2>&1; then
    fail "missing release asset was accepted"
fi

curl_args_file="${TEST_ROOT}/curl-args"
curl() {
    printf '%s\n' "$@" > "$curl_args_file"
    printf '%s' "$release_json"
}
GITHUB_AUTH_TOKEN=private-read-token
github_api_get "https://api.github.com/repos/example/private-release/releases/latest" >/dev/null
grep -Fxq 'Authorization: Bearer private-read-token' "$curl_args_file" \
    || fail "private release token was not sent"
grep -Fxq 'Accept: application/vnd.github+json' "$curl_args_file" \
    || fail "GitHub JSON media type was not sent"

github_download_asset \
    "https://api.github.com/repos/example/private-release/releases/assets/10" \
    "${TEST_ROOT}/downloaded-asset"
grep -Fxq 'Accept: application/octet-stream' "$curl_args_file" \
    || fail "GitHub asset media type was not sent"
grep -Fxq 'Authorization: Bearer private-read-token' "$curl_args_file" \
    || fail "private asset token was not sent"

archive_name=sub2api_1.2.3-entangled.4_linux_amd64.tar.gz
archive_path="${TEST_ROOT}/${archive_name}"
checksum_path="${TEST_ROOT}/checksums.txt"
printf 'verified release payload' > "$archive_path"
archive_hash=$(calculate_sha256 "$archive_path")
printf '%s  %s\n' "$archive_hash" "$archive_name" > "$checksum_path"
verify_release_checksum "$archive_path" "$checksum_path" "$archive_name" \
    || fail "valid checksum was rejected"

printf '%064d  %s\n' 0 "$archive_name" > "$checksum_path"
if verify_release_checksum "$archive_path" "$checksum_path" "$archive_name"; then
    fail "checksum mismatch was accepted"
fi
printf '%s  other.tar.gz\n' "$archive_hash" > "$checksum_path"
if verify_release_checksum "$archive_path" "$checksum_path" "$archive_name"; then
    fail "missing checksum entry was accepted"
fi

printf 'install script release-source tests passed\n'
