#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="$ROOT_DIR/bin"
TMP_DIR="$ROOT_DIR/tmp"
RUNTIME_DIR="$TMP_DIR/runtime"

cleanup() {
    rm -rf "$RUNTIME_DIR"
    rm -f "$BUNDLE_OUTPUT"
    rm -f "$BUNDLE_ASSET"
}

trap cleanup EXIT

BUNDLE_BUILDER="$OUTPUT_DIR/bundle-builder"
TOOL_FETCHER="$OUTPUT_DIR/tool-fetch"
BUNDLE_OUTPUT="$TMP_DIR/runtime.bundle"
BUNDLE_ASSET="$ROOT_DIR/internal/bundle/runtime.bundle"
UNISHELL_BINARY="$OUTPUT_DIR/unishell"
ASSET_EXCLUDES=(
    "tools/"
)

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

stage_runtime_assets() {
    local source="$ROOT_DIR/assets"
    local destination="$RUNTIME_DIR"
    local entry relative excluded

    while IFS= read -r -d '' entry; do
        relative="${entry#"$source"/}"
        excluded=false

        for pattern in "${ASSET_EXCLUDES[@]}"; do
            if [[ "$pattern" == */ ]]; then
                if [[ "$relative" == "${pattern%/}" || "$relative" == "${pattern%/}"/* ]]; then
                    excluded=true
                    break
                fi
            elif [[ "$relative" == "$pattern" ]]; then
                excluded=true
                break
            fi
        done

        if [[ "$excluded" == true ]]; then
            continue
        fi

        if [[ -d "$entry" ]]; then
            mkdir -p "$destination/$relative"
            continue
        fi

        mkdir -p "$destination/$(dirname "$relative")"
        cp -a "$entry" "$destination/$relative"
    done < <(find "$source" -mindepth 1 -print0)
}

echo "==> Preparing runtime assets"

rm -rf "$RUNTIME_DIR"
mkdir -p "$RUNTIME_DIR"

stage_runtime_assets

echo "==> Generating runtime bundle"

"$BUNDLE_BUILDER" \
    -input "$RUNTIME_DIR" \
    -output "$BUNDLE_OUTPUT" \

    if [[ ! -s "$BUNDLE_OUTPUT" ]]; then
        echo "error: bundle builder did not create $BUNDLE_OUTPUT" >&2
        exit 1
    fi

    rm -f "$BUNDLE_ASSET"
    cp "$BUNDLE_OUTPUT" "$BUNDLE_ASSET"

    if [[ ! -s "$BUNDLE_ASSET" ]]; then
        echo "error: failed to prepare embedded bundle $BUNDLE_ASSET" >&2
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
