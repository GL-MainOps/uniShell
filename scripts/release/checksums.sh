#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_DIR="$ROOT_DIR/bin"
OUTPUT_FILE="${1:-$ROOT_DIR/SHA256SUMS}"

mapfile -t binaries < <(
    find "$BIN_DIR" -maxdepth 1 -type f -name 'unishell-*' -print | sort
)

if [[ ${#binaries[@]} -eq 0 ]]; then
    echo "error: no release binaries found in $BIN_DIR" >&2
    exit 1
fi

rm -f "$OUTPUT_FILE"

for binary in "${binaries[@]}"; do
    (
        cd "$BIN_DIR"
        sha256sum "$(basename "$binary")"
    )
done > "$OUTPUT_FILE"

printf '%s\n' "Generated $OUTPUT_FILE"
