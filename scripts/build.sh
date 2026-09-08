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
TOOL_FETCHER="$OUTPUT_DIR/tool-fetch"
BUNDLE_OUTPUT="$TMP_DIR/runtime.bundle"
BUNDLE_SOURCE="$ROOT_DIR/internal/bundle/generated_bundle.go"
UNISHELL_BINARY="$OUTPUT_DIR/unishell"

BUILD_COMMIT="$(git rev-parse --short=7 HEAD)"
BUILD_DATE="$(date -u +%Y%m%d)"
BUILD_VERSION="${BUILD_COMMIT}-${BUILD_DATE}"

SKIP_FETCH=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --skip-fetch)
            SKIP_FETCH=true
            shift
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "error: unknown build option: $1" >&2
            exit 1
            ;;
    esac
done

if [[ $# -gt 0 ]]; then
    echo "error: unexpected build argument: $1" >&2
    exit 1
fi

mkdir -p "$OUTPUT_DIR" "$TMP_DIR"

cd "$ROOT_DIR"

if [[ "$SKIP_FETCH" == false ]]; then
    echo "==> Building tool-fetch"

    go build \
        -trimpath \
        -ldflags "-s -w" \
        -o "$TOOL_FETCHER" \
        ./cmd/tool-fetch

    echo "==> Fetching runtime tools"

    "$TOOL_FETCHER"
else
    echo "==> Skipping runtime tool acquisition"
fi

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
    -ldflags "-s -w -X main.version=$BUILD_VERSION -X main.commit=$BUILD_COMMIT" \
    -o "$UNISHELL_BINARY" \
    ./cmd/unishell

echo "==> Build complete"
echo "unishell:       $UNISHELL_BINARY"
echo "bundle-builder: $BUNDLE_BUILDER"
echo "bundle:         $BUNDLE_OUTPUT"
