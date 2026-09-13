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
    "bin/"
    "tools/"
)

BUILD_COMMIT="$(git rev-parse --short=7 HEAD)"
BUILD_DATE="$(date -u +%Y%m%d)"
BUILD_VERSION="${BUILD_COMMIT}-${BUILD_DATE}"

SKIP_FETCH=false
PROFILE_LIST=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --skip-fetch)
            SKIP_FETCH=true
            shift
            ;;
        --profile)
            if [[ -n "$PROFILE_LIST" ]]; then
                echo "error: --profile may only be specified once" >&2
                exit 1
            fi

            if [[ $# -lt 2 || -z "$2" ]]; then
                echo "error: --profile requires a comma-separated profile list" >&2
                exit 1
            fi

            PROFILE_LIST="$2"
            shift 2
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

echo "==> Building tool-fetch"

go build \
    -trimpath \
    -ldflags "-s -w" \
    -o "$TOOL_FETCHER" \
    ./cmd/tool-fetch

if [[ "$SKIP_FETCH" == true ]]; then
    echo "==> Skipping runtime tool acquisition"
elif [[ -z "$PROFILE_LIST" ]]; then
    echo "==> Fetching runtime tools"

    "$TOOL_FETCHER"
fi

echo "==> Building bundle-builder"

go build \
    -trimpath \
    -ldflags "-s -w" \
    -o "$BUNDLE_BUILDER" \
    ./cmd/bundle-builder

stage_runtime_assets() {
    local destination="$1"
    local source="$ROOT_DIR/assets"
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

split_profiles() {
    local value="$1"
    local profile

    IFS=',' read -r -a profiles <<< "$value"

    for profile in "${profiles[@]}"; do
        profile="${profile#"${profile%%[![:space:]]*}"}"
        profile="${profile%"${profile##*[![:space:]]}"}"

        if [[ -z "$profile" ]]; then
            echo "error: profile list contains an empty profile name" >&2
            return 1
        fi

        if [[ "$profile" == "common" ]]; then
            echo 'error: profile "common" is reserved and cannot be selected directly' >&2
            return 1
        fi

        printf '%s\n' "$profile"
    done
}

build_profile() {
    local profile="$1"
    local profile_runtime_dir="$TMP_DIR/runtime-$profile"
    local profile_bundle_output="$TMP_DIR/runtime-$profile.bundle"
    local profile_bundle_asset="$ROOT_DIR/internal/bundle/runtime.bundle"
    local profile_binary="$OUTPUT_DIR/unishell-$profile"

    cleanup_profile() {
        rm -rf "$profile_runtime_dir"
        rm -f "$profile_bundle_output"
        rm -f "$profile_bundle_asset"
    }

    trap cleanup_profile RETURN

    echo "==> Building profile: $profile"

    rm -rf "$profile_runtime_dir"
    mkdir -p "$profile_runtime_dir"

    stage_runtime_assets "$profile_runtime_dir"

    if [[ "$SKIP_FETCH" == false ]]; then
        echo "==> Fetching tools for profile: $profile"

        "$TOOL_FETCHER" \
            -profile "$profile"
    fi

    echo "==> Materializing cached tools for profile: $profile"

    "$TOOL_FETCHER" \
        -tools-dir "$ROOT_DIR/assets/tools" \
        -source-dir "$ROOT_DIR/assets/bin" \
        -output-dir "$profile_runtime_dir/bin" \
        -profile "$profile" \
        -cached

    echo "==> Generating bundle for profile: $profile"

    "$BUNDLE_BUILDER" \
        -input "$profile_runtime_dir" \
        -output "$profile_bundle_output"

    if [[ ! -s "$profile_bundle_output" ]]; then
        echo "error: bundle builder did not create $profile_bundle_output" >&2
        return 1
    fi

    rm -f "$profile_bundle_asset"
    cp "$profile_bundle_output" "$profile_bundle_asset"

    if [[ ! -s "$profile_bundle_asset" ]]; then
        echo "error: failed to prepare embedded bundle $profile_bundle_asset" >&2
        return 1
    fi

    echo "==> Building unishell-$profile"

    go build \
        -trimpath \
        -tags unishell_bundle \
        -ldflags "-s -w -X main.version=$BUILD_VERSION -X main.commit=$BUILD_COMMIT" \
        -o "$profile_binary" \
        ./cmd/unishell

    echo "unishell-$profile: $profile_binary"
}

if [[ -n "$PROFILE_LIST" ]]; then
    mapfile -t profiles < <(split_profiles "$PROFILE_LIST")

    for profile in "${profiles[@]}"; do
        build_profile "$profile"
    done

    echo "==> Profile builds complete"

    for profile in "${profiles[@]}"; do
        echo "unishell-$profile: $OUTPUT_DIR/unishell-$profile"
    done

    exit 0
fi

echo "==> Preparing runtime assets"

rm -rf "$RUNTIME_DIR"
mkdir -p "$RUNTIME_DIR"

stage_runtime_assets "$RUNTIME_DIR"

echo "==> Generating runtime bundle"

"$BUNDLE_BUILDER" \
    -input "$RUNTIME_DIR" \
    -output "$BUNDLE_OUTPUT"

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
