#!/usr/bin/env bash

set -euo pipefail

GITLAB_RELEASES_URL="https://gitlab.com/mainops/uniShell/-/releases"
GITLAB_LATEST_RELEASE_URL="$GITLAB_RELEASES_URL/permalink/latest"

GITLAB_RELEASE_DOWNLOADS_URL=""

NEXUS_BASE_URL="https://ntgrepo.ntgcloud.net"
NEXUS_REPOSITORY="raw"
NEXUS_NAMESPACE="lesscache"

CREDENTIALS_FILE="$HOME/.local/state/.repo"
STAGING_DIRECTORY="$HOME/.local/state/.lesscache"

require_command() {
    local command="$1"

    if ! command -v "$command" >/dev/null 2>&1; then
        echo "error: required command not found: $command" >&2
        exit 1
    fi
}

load_credentials() {
    if [[ ! -f "$CREDENTIALS_FILE" ]]; then
        echo "error: credentials file not found: $CREDENTIALS_FILE" >&2
        exit 1
    fi

    if [[ "$(wc -l < "$CREDENTIALS_FILE")" -ne 1 ]]; then
        echo "error: credentials file must contain exactly one line" >&2
        exit 1
    fi

    local credentials
    credentials="$(<"$CREDENTIALS_FILE")"

    if [[ "$credentials" != *:* ]]; then
        echo "error: credentials file must use the format admin:password" >&2
        exit 1
    fi

    NEXUS_USERNAME="${credentials%%:*}"
    NEXUS_PASSWORD="${credentials#*:}"

    if [[ -z "$NEXUS_USERNAME" || -z "$NEXUS_PASSWORD" ]]; then
        echo "error: credentials file must contain a username and password" >&2
        exit 1
    fi
}

prepare_staging_directory() {
    mkdir -p "$STAGING_DIRECTORY"

    find "$STAGING_DIRECTORY" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
}

cleanup_staging_directory() {
    if [[ -d "$STAGING_DIRECTORY" ]]; then
        find "$STAGING_DIRECTORY" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
    fi
}

get_latest_gitlab_release() {
    local redirect_url

    redirect_url="$(
        curl \
            --fail \
            --silent \
            --show-error \
            --output /dev/null \
            --write-out '%{redirect_url}' \
            "$GITLAB_LATEST_RELEASE_URL"
    )"

    if [[ -z "$redirect_url" ]]; then
        echo "error: GitLab latest release did not provide a redirect URL" >&2
        exit 1
    fi

    GITLAB_RELEASE_TAG="${redirect_url##*/}"

    if [[ ! "$GITLAB_RELEASE_TAG" =~ ^v[0-9]+(\.[0-9]+)*$ ]]; then
        echo "error: GitLab latest release redirect does not contain a valid release tag: $GITLAB_RELEASE_TAG" >&2
        exit 1
    fi
}

get_gitlab_release_manifest() {
    local manifest_url
    local manifest_path
    local line

    manifest_path="$STAGING_DIRECTORY/SHA256SUMS"
    manifest_url="$GITLAB_RELEASES_URL/$GITLAB_RELEASE_TAG/downloads/SHA256SUMS"

    prepare_staging_directory

    curl \
        --fail \
        --silent \
        --show-error \
        --location \
        --output "$manifest_path" \
        "$manifest_url"

    if [[ ! -s "$manifest_path" ]]; then
        echo "error: GitLab release SHA256SUMS manifest is empty" >&2
        exit 1
    fi

    GITLAB_RELEASE_DOWNLOADS_URL="$GITLAB_RELEASES_URL/$GITLAB_RELEASE_TAG/downloads"
    GITLAB_BINARY_NAMES=()

    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ -z "$line" ]]; then
            continue
        fi

        if [[ ! "$line" =~ ^[0-9a-fA-F]{64}[[:space:]]{1,2}(unishell-[^[:space:]]+)$ ]]; then
            echo "error: GitLab SHA256SUMS contains an invalid entry: $line" >&2
            exit 1
        fi

        GITLAB_BINARY_NAMES+=("${BASH_REMATCH[1]}")
    done < "$manifest_path"

    if [[ ${#GITLAB_BINARY_NAMES[@]} -eq 0 ]]; then
        echo "error: GitLab SHA256SUMS does not contain any unishell-* binaries" >&2
        exit 1
    fi
}

get_nexus_binary_name() {
    local gitlab_binary_name="$1"

    printf 'lesscache-%s\n' "${gitlab_binary_name#unishell-}"
}

download_and_verify_artifacts() {
    local binary_name
    local expected_checksum
    local artifact_url
    local artifact_path

    echo "==> Downloading and verifying GitLab release artifacts"

    while IFS= read -r binary_name; do
        expected_checksum="$(
            awk -v name="$binary_name" '$2 == name { print $1; exit }' \
                "$STAGING_DIRECTORY/SHA256SUMS"
        )"

        if [[ -z "$expected_checksum" ]]; then
            echo "error: no checksum found for GitLab release artifact: $binary_name" >&2
            exit 1
        fi

        artifact_url="$GITLAB_RELEASE_DOWNLOADS_URL/$binary_name"
        artifact_path="$STAGING_DIRECTORY/$binary_name"

        echo "Downloading: $binary_name"

        curl \
            --fail \
            --silent \
            --show-error \
            --location \
            --output "$artifact_path" \
            "$artifact_url"

        printf '%s  %s\n' "$expected_checksum" "$artifact_path" |
            sha256sum --check --status

        echo "Verified:   $binary_name"
    done < <(printf '%s\n' "${GITLAB_BINARY_NAMES[@]}")
}

upload_artifacts_to_nexus() {
    local binary_name
    local nexus_binary_name
    local artifact_path
    local batch_count=0
    local -a upload_args=()

    echo "==> Uploading verified artifacts to Nexus"

    while IFS= read -r binary_name; do
        artifact_path="$STAGING_DIRECTORY/$binary_name"
        nexus_binary_name="$(get_nexus_binary_name "$binary_name")"

        if [[ ! -f "$artifact_path" ]]; then
            echo "error: staged artifact not found: $artifact_path" >&2
            exit 1
        fi

        upload_args+=(
            --form "raw.asset$((batch_count + 1))=@$artifact_path"
            --form "raw.asset$((batch_count + 1)).filename=$nexus_binary_name"
        )

        batch_count=$((batch_count + 1))

        if (( batch_count == 3 )); then
            curl \
                --fail \
                --silent \
                --show-error \
                --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                --request POST \
                --url "$NEXUS_BASE_URL/service/rest/v1/components?repository=$NEXUS_REPOSITORY" \
                --form "raw.directory=$NEXUS_NAMESPACE" \
                "${upload_args[@]}"

            echo "Uploaded batch: $batch_count artifact(s)"

            upload_args=()
            batch_count=0
        fi
    done < <(printf '%s\n' "${GITLAB_BINARY_NAMES[@]}")

    if (( batch_count > 0 )); then
        curl \
            --fail \
            --silent \
            --show-error \
            --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
            --request POST \
            --url "$NEXUS_BASE_URL/service/rest/v1/components?repository=$NEXUS_REPOSITORY" \
            --form "raw.directory=$NEXUS_NAMESPACE" \
            "${upload_args[@]}"

        echo "Uploaded batch: $batch_count artifact(s)"
    fi
}

verify_nexus_artifacts() {
    local continuation=""
    local response
    local binary_name
    local nexus_binary_name
    local expected_checksum
    local asset_path
    local asset_id
    local download_url
    local match_count
    local actual_checksum

    echo "==> Verifying uploaded artifacts in Nexus"

    while IFS= read -r binary_name; do
        nexus_binary_name="$(get_nexus_binary_name "$binary_name")"
        expected_checksum="$(
            awk -v name="$binary_name" '$2 == name { print $1; exit }' \
                "$STAGING_DIRECTORY/SHA256SUMS"
        )"

        if [[ -z "$expected_checksum" ]]; then
            echo "error: no checksum found for GitLab release artifact: $binary_name" >&2
            exit 1
        fi

        asset_path="$NEXUS_NAMESPACE/$nexus_binary_name"
        continuation=""
        match_count=0
        asset_id=""
        download_url=""

        while :; do
            if [[ -n "$continuation" ]]; then
                response="$(
                    curl \
                        --fail \
                        --silent \
                        --show-error \
                        --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                        --get \
                        --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                        --data-urlencode "repository=$NEXUS_REPOSITORY" \
                        --data-urlencode "continuationToken=$continuation"
                )"
            else
                response="$(
                    curl \
                        --fail \
                        --silent \
                        --show-error \
                        --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                        --get \
                        --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                        --data-urlencode "repository=$NEXUS_REPOSITORY"
                )"
            fi

            while IFS=$'\t' read -r candidate_id candidate_path candidate_url; do
                if [[ "$candidate_path" == "$asset_path" ]]; then
                    match_count=$((match_count + 1))
                    asset_id="$candidate_id"
                    download_url="$candidate_url"
                fi
            done < <(
                jq -r '
                    .items[]? |
                    [
                        (.id // ""),
                        (.path // ""),
                        (.downloadUrl // "")
                    ] |
                    @tsv
                ' <<< "$response"
            )

            continuation="$(jq -r '.continuationToken // empty' <<< "$response")"

            if [[ -z "$continuation" ]]; then
                break
            fi
        done

        if (( match_count == 0 )); then
            echo "error: Nexus artifact not found: $asset_path" >&2
            exit 1
        fi

        if (( match_count > 1 )); then
            echo "error: Nexus contains multiple assets at path: $asset_path" >&2
            exit 1
        fi

        if [[ -z "$asset_id" || -z "$download_url" ]]; then
            echo "error: Nexus asset metadata is incomplete for: $asset_path" >&2
            exit 1
        fi

        actual_checksum="$(
            curl \
                --fail \
                --silent \
                --show-error \
                --location \
                --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                "$download_url" |
                sha256sum |
                awk '{ print $1 }'
        )"

        if [[ "$actual_checksum" != "$expected_checksum" ]]; then
            echo "error: Nexus artifact checksum mismatch: $binary_name" >&2
            echo "expected: $expected_checksum" >&2
            echo "actual:   $actual_checksum" >&2
            exit 1
        fi

        echo "Verified:   $binary_name"
    done < <(printf '%s\n' "${GITLAB_BINARY_NAMES[@]}")
}

upload_release_marker() {

    if [[ ! -f "$STAGING_DIRECTORY/SHA256SUMS" ]]; then
        echo "error: staged SHA256SUMS not found: $STAGING_DIRECTORY/SHA256SUMS" >&2
        exit 1
    fi

    echo "==> Publishing Nexus release marker: $GITLAB_RELEASE_TAG"

    curl \
        --fail \
        --silent \
        --show-error \
        --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
        --request POST \
        --url "$NEXUS_BASE_URL/service/rest/v1/components?repository=$NEXUS_REPOSITORY" \
        --form "raw.directory=$NEXUS_NAMESPACE" \
        --form "raw.asset1=@$STAGING_DIRECTORY/SHA256SUMS" \
        --form "raw.asset1.filename=$GITLAB_RELEASE_TAG"

    echo "Published marker: $GITLAB_RELEASE_TAG"
}

verify_release_marker() {
    local marker_path="$NEXUS_NAMESPACE/$GITLAB_RELEASE_TAG"
    local continuation=""
    local response
    local candidate_path
    local candidate_url
    local marker_download_url=""
    local match_count=0
    local actual_checksum
    local expected_checksum

    expected_checksum="$(
        sha256sum "$STAGING_DIRECTORY/SHA256SUMS" |
            awk '{ print $1 }'
    )"

    echo "==> Verifying Nexus release marker: $GITLAB_RELEASE_TAG"

    while :; do
        if [[ -n "$continuation" ]]; then
            response="$(
                curl \
                    --fail \
                    --silent \
                    --show-error \
                    --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                    --get \
                    --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                    --data-urlencode "repository=$NEXUS_REPOSITORY" \
                    --data-urlencode "continuationToken=$continuation"
            )"
        else
            response="$(
                curl \
                    --fail \
                    --silent \
                    --show-error \
                    --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                    --get \
                    --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                    --data-urlencode "repository=$NEXUS_REPOSITORY"
            )"
        fi

        while IFS=$'\t' read -r candidate_path candidate_url; do
            if [[ "$candidate_path" == "$marker_path" ]]; then
                match_count=$((match_count + 1))
                marker_download_url="$candidate_url"
            fi
        done < <(
            jq -r '
                .items[]? |
                [
                    (.path // ""),
                    (.downloadUrl // "")
                ] |
                @tsv
            ' <<< "$response"
        )

        continuation="$(jq -r '.continuationToken // empty' <<< "$response")"

        if [[ -z "$continuation" ]]; then
            break
        fi
    done

    if (( match_count == 0 )); then
        echo "error: Nexus release marker not found: $marker_path" >&2
        exit 1
    fi

    if (( match_count > 1 )); then
        echo "error: Nexus contains multiple release markers at path: $marker_path" >&2
        exit 1
    fi

    if [[ -z "$marker_download_url" ]]; then
        echo "error: Nexus release marker metadata is incomplete: $marker_path" >&2
        exit 1
    fi

    actual_checksum="$(
        curl \
            --fail \
            --silent \
            --show-error \
            --location \
            --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
            "$marker_download_url" |
            sha256sum |
            awk '{ print $1 }'
    )"

    if [[ "$actual_checksum" != "$expected_checksum" ]]; then
        echo "error: Nexus release marker checksum mismatch: $GITLAB_RELEASE_TAG" >&2
        echo "expected: $expected_checksum" >&2
        echo "actual:   $actual_checksum" >&2
        exit 1
    fi

    echo "Verified marker: $GITLAB_RELEASE_TAG"
}

delete_old_release_markers() {
    local continuation=""
    local response
    local asset_id
    local asset_path
    local next_token
    local version
    local -a old_marker_ids=()

    echo "==> Removing older Nexus release markers"

    while :; do
        if [[ -n "$continuation" ]]; then
            response="$(
                curl \
                    --fail \
                    --silent \
                    --show-error \
                    --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                    --get \
                    --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                    --data-urlencode "repository=$NEXUS_REPOSITORY" \
                    --data-urlencode "continuationToken=$continuation"
            )"
        else
            response="$(
                curl \
                    --fail \
                    --silent \
                    --show-error \
                    --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                    --get \
                    --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                    --data-urlencode "repository=$NEXUS_REPOSITORY"
            )"
        fi

        while IFS=$'\t' read -r asset_id asset_path; do
            if [[ "$asset_path" != "$NEXUS_NAMESPACE"/v* ]]; then
                continue
            fi

            version="${asset_path#"$NEXUS_NAMESPACE"/}"

            if [[ "$version" == "$GITLAB_RELEASE_TAG" ]]; then
                continue
            fi

            if [[ "$(printf '%s\n' "$version" "$GITLAB_RELEASE_TAG" | sort -V | tail -n 1)" != "$GITLAB_RELEASE_TAG" ]]; then
                continue
            fi

            if [[ -z "$asset_id" ]]; then
                echo "error: Nexus release marker has no asset ID: $asset_path" >&2
                exit 1
            fi

            old_marker_ids+=("$asset_id")
            echo "Queued marker for deletion: $version"
        done < <(
            jq -r '
                .items[]? |
                [
                    (.id // ""),
                    (.path // "")
                ] |
                @tsv
            ' <<< "$response"
        )

        next_token="$(jq -r '.continuationToken // empty' <<< "$response")"

        if [[ -z "$next_token" ]]; then
            break
        fi

        continuation="$next_token"
    done

    for asset_id in "${old_marker_ids[@]}"; do
        echo "Deleting marker asset: $asset_id"

        curl \
            --fail \
            --silent \
            --show-error \
            --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
            --request DELETE \
            --url "$NEXUS_BASE_URL/service/rest/v1/assets/$asset_id"
    done
}

verify_release_markers_cleanup() {
    local continuation=""
    local response
    local asset_path
    local version
    local next_token
    local marker_count=0

    echo "==> Verifying Nexus release marker cleanup"

    while :; do
        if [[ -n "$continuation" ]]; then
            response="$(
                curl \
                    --fail \
                    --silent \
                    --show-error \
                    --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                    --get \
                    --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                    --data-urlencode "repository=$NEXUS_REPOSITORY" \
                    --data-urlencode "continuationToken=$continuation"
            )"
        else
            response="$(
                curl \
                    --fail \
                    --silent \
                    --show-error \
                    --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                    --get \
                    --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                    --data-urlencode "repository=$NEXUS_REPOSITORY"
            )"
        fi

        while IFS= read -r asset_path; do
            if [[ "$asset_path" != "$NEXUS_NAMESPACE"/v* ]]; then
                continue
            fi

            version="${asset_path#"$NEXUS_NAMESPACE"/}"

            if [[ "$version" != "$GITLAB_RELEASE_TAG" ]]; then
                echo "error: unexpected Nexus release marker remains: $asset_path" >&2
                exit 1
            fi

            marker_count=$((marker_count + 1))
        done < <(
            jq -r '.items[]?.path // empty' <<< "$response"
        )

        next_token="$(jq -r '.continuationToken // empty' <<< "$response")"

        if [[ -z "$next_token" ]]; then
            break
        fi

        continuation="$next_token"
    done

    if (( marker_count == 0 )); then
        echo "error: Nexus release marker missing after cleanup: $GITLAB_RELEASE_TAG" >&2
        exit 1
    fi

    if (( marker_count > 1 )); then
        echo "error: Nexus contains multiple release markers for current release: $GITLAB_RELEASE_TAG" >&2
        exit 1
    fi

    echo "Verified marker cleanup: only $GITLAB_RELEASE_TAG remains"
}

get_nexus_current_version() {
    local continuation=""
    local response
    local path
    local next_token
    local version

    NEXUS_CURRENT_VERSION=""

    while :; do
        if [[ -n "$continuation" ]]; then
            response="$(
                curl \
                    --fail \
                    --silent \
                    --show-error \
                    --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                    --get \
                    --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                    --data-urlencode "repository=$NEXUS_REPOSITORY" \
                    --data-urlencode "continuationToken=$continuation"
            )"
        else
            response="$(
                curl \
                    --fail \
                    --silent \
                    --show-error \
                    --user "$NEXUS_USERNAME:$NEXUS_PASSWORD" \
                    --get \
                    --url "$NEXUS_BASE_URL/service/rest/v1/assets" \
                    --data-urlencode "repository=$NEXUS_REPOSITORY"
            )"
        fi

        while IFS= read -r path; do
            if [[ "$path" == "$NEXUS_NAMESPACE"/v* ]]; then
                version="${path#"$NEXUS_NAMESPACE"/}"

                if [[ -z "$NEXUS_CURRENT_VERSION" ]] ||
                   [[ "$(printf '%s\n' "$NEXUS_CURRENT_VERSION" "$version" | sort -V | tail -n 1)" == "$version" ]]; then
                    NEXUS_CURRENT_VERSION="$version"
                fi
            fi
        done < <(jq -r '.items[]?.path // empty' <<< "$response")

        next_token="$(jq -r '.continuationToken // empty' <<< "$response")"

        if [[ -z "$next_token" ]]; then
            break
        fi

        continuation="$next_token"
    done
}

compare_versions() {
    if [[ -z "$NEXUS_CURRENT_VERSION" ]]; then
        VERSION_STATE="initial"
        return
    fi

    if [[ "$GITLAB_RELEASE_TAG" == "$NEXUS_CURRENT_VERSION" ]]; then
        VERSION_STATE="same"
        return
    fi

    local newest
    newest="$(
        printf '%s\n' "$NEXUS_CURRENT_VERSION" "$GITLAB_RELEASE_TAG" |
            sort -V |
            tail -n 1
    )"

    if [[ "$newest" == "$GITLAB_RELEASE_TAG" ]]; then
        VERSION_STATE="newer"
    else
        VERSION_STATE="older"
    fi
}

main() {
    require_command curl
    require_command jq
    require_command sort
    require_command wc
    require_command sha256sum
    require_command awk
    require_command find

    load_credentials

    trap cleanup_staging_directory EXIT

    echo "==> Checking GitLab latest release"
    get_latest_gitlab_release
    echo "GitLab latest release: $GITLAB_RELEASE_TAG"

    echo "==> Checking Nexus release state"

    get_nexus_current_version

    if [[ -n "$NEXUS_CURRENT_VERSION" ]]; then
        echo "Nexus current release:  $NEXUS_CURRENT_VERSION"
    else
        echo "Nexus current release:  none"
    fi

    compare_versions

    case "$VERSION_STATE" in
        initial)
            echo "==> Nexus has no release marker; synchronization is required"
            echo "==> Checking GitLab release manifest"
            get_gitlab_release_manifest
            echo "GitLab checksum manifest: SHA256SUMS"
            printf '%s\n' "GitLab release binaries:"
            for binary_name in "${GITLAB_BINARY_NAMES[@]}"; do
                printf '%s\n' "  $binary_name"
            done
            download_and_verify_artifacts
            upload_artifacts_to_nexus
            verify_nexus_artifacts
            upload_release_marker
            verify_release_marker
            delete_old_release_markers
            verify_release_markers_cleanup
            ;;
        same)
            echo "==> Nexus is already synchronized with GitLab"
            ;;
        newer)
            echo "==> GitLab release $GITLAB_RELEASE_TAG is newer than Nexus release $NEXUS_CURRENT_VERSION"
            echo "==> Checking GitLab release manifest"
            get_gitlab_release_manifest
            echo "GitLab checksum manifest: SHA256SUMS"
            printf '%s\n' "GitLab release binaries:"
            for binary_name in "${GITLAB_BINARY_NAMES[@]}"; do
                printf '%s\n' "  $binary_name"
            done
            download_and_verify_artifacts
            upload_artifacts_to_nexus
            verify_nexus_artifacts
            upload_release_marker
            verify_release_marker
            delete_old_release_markers
            verify_release_markers_cleanup
            ;;
        older)
            echo "error: GitLab release $GITLAB_RELEASE_TAG is older than Nexus release $NEXUS_CURRENT_VERSION" >&2
            exit 1
            ;;
    esac
}

main "$@"
