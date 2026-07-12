#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
entrypoint="$script_dir/docker-entrypoint.sh"
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/sub2api-entrypoint-test.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

image_binary="$tmp_dir/image-sub2api"
runtime_dir="$tmp_dir/data/runtime"
runtime_binary="$runtime_dir/sub2api"

write_fake_binary() {
    target=$1
    label=$2
    mkdir -p "$(dirname -- "$target")"
    printf '#!/bin/sh\nprintf "%%s\\n" "%s"\n' "$label" > "$target"
    chmod 0755 "$target"
}

run_entrypoint() {
    policy=$1
    SUB2API_IMAGE_BINARY="$image_binary" \
    SUB2API_RUNTIME_DIR="$runtime_dir" \
    SUB2API_RUNTIME_SEED_POLICY="$policy" \
    sh "$entrypoint"
}

write_fake_binary "$image_binary" "image-v1"

first_output=$(run_entrypoint if-missing)
[ "$first_output" = "image-v1" ]
[ -x "$runtime_binary" ]

# Simulate a successful dashboard update replacing the persistent executable.
write_fake_binary "$runtime_binary" "dashboard-v2"
preserved_output=$(run_entrypoint if-missing)
[ "$preserved_output" = "dashboard-v2" ]

# Recovery mode intentionally restores the image seed.
reset_output=$(run_entrypoint always)
[ "$reset_output" = "image-v1" ]

# Immutable mode bypasses the persistent runtime entirely.
write_fake_binary "$runtime_binary" "dashboard-v3"
immutable_output=$(run_entrypoint never)
[ "$immutable_output" = "image-v1" ]

printf 'docker-entrypoint persistent runtime tests passed\n'
