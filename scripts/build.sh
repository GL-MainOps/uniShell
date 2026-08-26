#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="$ROOT_DIR/bin"
TMP_DIR="$ROOT_DIR/tmp"
RUNTIME_DIR="$TMP_DIR/runtime"

cleanup() {
    rm -rf "$RUNTIME_DIR"
}

trap cleanup EXIT

BUNDLE_BUILDER="$OUTPUT_DIR/bundle-builder"
BUNDLE_OUTPUT="$TMP_DIR/runtime.bundle"
BUNDLE_SOURCE="$ROOT_DIR/internal/bundle/generated_bundle.go"
UNISHELL_BINARY="$OUTPUT_DIR/unishell"

mkdir -p "$OUTPUT_DIR" "$TMP_DIR"

cd "$ROOT_DIR"

echo "==> Building bundle-builder"

go build \
    -trimpath \
    -ldflags "-s -w" \
    -o "$BUNDLE_BUILDER" \
    ./cmd/bundle-builder

echo "==> Preparing runtime assets"

rm -rf "$RUNTIME_DIR"
mkdir -p "$RUNTIME_DIR"

cp -a "$ROOT_DIR/assets/." "$RUNTIME_DIR/"

echo "==> Generating runtime bundle"

"$BUNDLE_BUILDER" \
    -input "$RUNTIME_DIR" \
    -output "$BUNDLE_OUTPUT" \
    -generate "$BUNDLE_SOURCE"

if [[ ! -s "$BUNDLE_OUTPUT" ]]; then
    echo "error: bundle builder did not create $BUNDLE_OUTPUT" >&2
    exit 1
fi

if [[ ! -s "$BUNDLE_SOURCE" ]]; then
    echo "error: bundle builder did not generate $BUNDLE_SOURCE" >&2
    exit 1
fi

echo "==> Building unishell"

go build \
    -trimpath \
    -tags unishell_bundle \
    -ldflags "-s -w" \
    -o "$UNISHELL_BINARY" \
    ./cmd/unishell

echo "==> Build complete"
echo "unishell:       $UNISHELL_BINARY"
echo "bundle-builder: $BUNDLE_BUILDER"
echo "bundle:         $BUNDLE_OUTPUT"
