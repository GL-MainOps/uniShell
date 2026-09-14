#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
TAG="${1:?error: release tag is required}"
CHECKSUMS_FILE="${2:?error: checksum file is required}"
OUTPUT_FILE="${3:-$ROOT_DIR/RELEASE_NOTES.md}"

cd "$ROOT_DIR"

if [[ "$TAG" != v* ]]; then
    echo "error: release tag must start with 'v': $TAG" >&2
    exit 1
fi

if ! git rev-parse --verify --quiet "$TAG^{commit}" >/dev/null; then
    echo "error: release tag '$TAG' does not exist" >&2
    exit 1
fi

previous_tag="$(
    git tag --list 'v*' --sort=-version:refname |
        awk -v tag="$TAG" '$0 != tag { print; exit }'
)"

{
    printf '%s\n\n' "# uniShell $TAG"
    printf '%s\n' '## What'\''s Changed'

    if [[ -n "$previous_tag" ]]; then
        git log --pretty=format:'- %s (%h)' "$previous_tag..$TAG"
    else
        git log --pretty=format:'- %s (%h)' "$TAG"
    fi

    printf '\n\n%s\n\n' '## Assets'

    while IFS= read -r binary; do
        printf '%s\n' "- \`$(basename "$binary")\`"
    done < <(
        find "$ROOT_DIR/bin" -maxdepth 1 -type f -name 'unishell-*' -printf '%f\n' | sort
    )

    printf '\n%s\n\n' '## SHA256 Checksums'
    printf '%s\n' '```text'
    cat "$CHECKSUMS_FILE"
    printf '%s\n' '```'
} > "$OUTPUT_FILE"

printf '%s\n' "Generated $OUTPUT_FILE"
