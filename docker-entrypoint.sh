#!/bin/sh
set -eu

# Named volumes are initially mounted as root. Prepare the database directory
# before dropping privileges for both extraction and application execution.
mkdir -p "${DB_PATH}"
chown -R brasilcep:brasilcep "${DB_PATH}"

# Extract on the container's writable layer so Badger can update its files.
# --sparse recreates the large value-log holes without allocating them on disk.
if [ ! -f "${DB_PATH}/MANIFEST" ]; then
    if [ -n "${DATA_DNE_EXTERNAL:-}" ]; then
        archive="$(mktemp /tmp/data-dne.tar.gz.XXXXXX)"
        trap 'rm -f "${archive}"' EXIT HUP INT TERM

        echo "Database not found at ${DB_PATH}; downloading external database"
        wget -T 60 -O "${archive}" "${DATA_DNE_EXTERNAL}"
        echo "Database archive downloaded ($(wc -c < "${archive}") bytes); extracting"
        chown brasilcep:brasilcep "${archive}"

        su-exec brasilcep tar --sparse -xzf "${archive}" --strip-components=1 -C "${DB_PATH}"

        if [ ! -f "${DB_PATH}/MANIFEST" ]; then
            echo "External database archive does not contain data/MANIFEST" >&2
            exit 1
        fi

        rm -f "${archive}"
        trap - EXIT HUP INT TERM
        echo "External database extracted successfully"
    fi
fi

exec su-exec brasilcep /app/wserver "$@"
