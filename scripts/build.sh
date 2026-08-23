#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="$ROOT_DIR/bin"

mkdir -p "$OUTPUT_DIR"

cd "$ROOT_DIR"

go build \
    -trimpath \
    -ldflags "-s -w" \
    -o "$OUTPUT_DIR/unishell" \
    ./cmd/unishell

echo "Built: $OUTPUT_DIR/unishell"
