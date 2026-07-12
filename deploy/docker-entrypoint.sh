#!/bin/sh
set -e

image_binary="${SUB2API_IMAGE_BINARY:-/app/sub2api}"
runtime_dir="${SUB2API_RUNTIME_DIR:-/app/data/runtime}"
runtime_binary="${runtime_dir}/sub2api"
seed_policy="${SUB2API_RUNTIME_SEED_POLICY:-if-missing}"

# The image binary is an immutable bootstrap seed. The executable that the
# online updater replaces lives in the persistent data volume, so an update
# survives process restarts and Docker container recreation.
seed_runtime_binary() {
    case "$seed_policy" in
        never)
            runtime_binary="$image_binary"
            return
            ;;
        if-missing)
            if [ -x "$runtime_binary" ]; then
                return
            fi
            ;;
        always)
            ;;
        *)
            echo "invalid SUB2API_RUNTIME_SEED_POLICY: $seed_policy (expected if-missing, always, or never)" >&2
            exit 1
            ;;
    esac

    if [ ! -x "$image_binary" ]; then
        echo "image seed binary is missing or not executable: $image_binary" >&2
        exit 1
    fi

    mkdir -p "$runtime_dir"
    seed_tmp="${runtime_binary}.seed.$$"
    rm -f "$seed_tmp"
    cp "$image_binary" "$seed_tmp"
    chmod 0755 "$seed_tmp"
    mv -f "$seed_tmp" "$runtime_binary"
}

# Fix persistent directory permissions when running as root. Docker named
# volumes / host bind-mounts may otherwise be owned by root.
if [ "$(id -u)" = "0" ]; then
    mkdir -p /app/data "$runtime_dir"
    # Use || true to avoid failure on read-only mounted files (e.g. config.yaml:ro)
    chown -R sub2api:sub2api /app/data 2>/dev/null || true
    chown -R sub2api:sub2api "$runtime_dir" 2>/dev/null || true
    # Re-invoke this script as sub2api so the flag-detection below
    # also runs under the correct user.
    exec su-exec sub2api "$0" "$@"
fi

seed_runtime_binary

if [ "$#" -eq 0 ]; then
    set -- "$runtime_binary"
fi

# Compatibility: if the first arg looks like a flag (e.g. --help),
# prepend the default binary so it behaves the same as the old
# ENTRYPOINT ["/app/sub2api"] style.
if [ "${1#-}" != "$1" ]; then
    set -- "$runtime_binary" "$@"
elif [ "$1" = "$image_binary" ]; then
    shift
    set -- "$runtime_binary" "$@"
fi

exec "$@"
