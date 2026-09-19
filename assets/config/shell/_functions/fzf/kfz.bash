# kfz.bash - fzf-driven Kubernetes operations for Bash
#
# Purpose:
#   Replace the high-frequency K9s workflows with a small Bash/fzf layer
#   around kubectl.  The main entry point is `kfz`.
#
# Examples:
#   kfz                         # choose a resource type, then browse it
#   kfz pods                    # browse pods in the current namespace
#   kfz pods -n observability   # browse pods in a specific namespace
#   kfz -n observability pods   # same, with namespace before the resource
#   kfz deploy                  # browse deployments
#   kfz ns                      # choose a namespace, then choose a resource
#   kfz ctx                     # choose/switch kubeconfig context
#
# Design notes:
#   - fzf matching is EXACT by default.
#   - Kubernetes previews are OPT-IN: the preview pane starts hidden and the
#     user's existing Ctrl-/ binding toggles it.
#   - Resource rows use private tab-delimited fields for actions, but the UI
#     receives a single pre-formatted display field. This prevents literal
#     "\\t" artifacts and keeps columns aligned even with long names.
#   - `kfz ns` is a namespace GATE, not a kubeconfig namespace switcher.
#   - `-n/--namespace` is accepted whenever a namespaced resource can use it.
#
# Requirements:
#   bash, kubectl, fzf
#
# Optional presentation tools:
#   bat (preferred for YAML/text), cat (fallback)
#   wl-copy, xclip, xsel, or pbcopy (clipboard)

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

_kfz__require_commands() {
    local missing=()
    local cmd

    for cmd in kubectl fzf; do
        command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
    done

    if ((${#missing[@]})); then
        printf 'kfz: missing command(s): %s\n' "${missing[*]}" >&2
        return 127
    fi
}

_kfz__pause() {
    printf '\nPress Enter to return to fzf...' >&2
    IFS= read -r _ </dev/tty
}

# Render stdin as preview content. bat is preferred because it handles paging,
# ANSI output, and syntax highlighting cleanly inside fzf. cat is the safe
# fallback because a pager such as less would take over the preview terminal.
_kfz__preview_render() {
    local language=${1:-text}

    if command -v bat >/dev/null 2>&1; then
        # Do not pass --language=text: `text` is not a portable bat language
        # identifier and can make bat exit without producing any output.
        if [[ $language == text ]]; then
            bat \
                --style=plain \
                --color=always \
                --paging=never \
                2>/dev/null
        else
            bat \
                --style=plain \
                --color=always \
                --paging=never \
                --language="$language" \
                2>/dev/null
        fi
    else
        cat
    fi
}

# Non-paged renderer used for output that should return immediately.
# Previews use _kfz__preview_render; full-screen actions use _kfz__page_file.
_kfz__render() {
    local language=${1:-text}

    if command -v bat >/dev/null 2>&1; then
        # Generic kubectl output should not force an invalid language name.
        if [[ $language == text ]]; then
            bat \
                --style=plain \
                --color=always \
                --paging=never \
                2>/dev/null
        else
            bat \
                --style=plain \
                --color=always \
                --paging=never \
                --language="$language" \
                2>/dev/null
        fi
    else
        cat
    fi
}

# Page full-screen action output such as Enter/Describe.
#
# The output is first written to a temporary file because bat's interactive
# pager is most reliable when bat receives an actual file rather than a pipe.
# Fallback order: bat -> less -R -> cat.
_kfz__page_file() {
    local language=${1:-text}
    local file=${2:-}

    [[ -r $file ]] || return 1

    if command -v bat >/dev/null 2>&1; then
        if [[ $language == text ]]; then
            bat \
                --style=plain \
                --color=always \
                --paging=always \
                "$file"
        else
            bat \
                --style=plain \
                --color=always \
                --paging=always \
                --language="$language" \
                "$file"
        fi
    elif command -v less >/dev/null 2>&1; then
        less -R "$file"
    else
        cat "$file"
    fi
}

_kfz__clip() {
    local value
    value=$(< /dev/stdin)

    if command -v wl-copy >/dev/null 2>&1; then
        printf '%s' "$value" | wl-copy
    elif command -v xclip >/dev/null 2>&1; then
        printf '%s' "$value" | xclip -selection clipboard
    elif command -v xsel >/dev/null 2>&1; then
        printf '%s' "$value" | xsel --clipboard --input
    elif command -v pbcopy >/dev/null 2>&1; then
        printf '%s' "$value" | pbcopy
    else
        printf '%s\n' "$value"
        return 1
    fi
}

# Resolve a kubectl resource/short-name to:
#   canonical-resource<TAB>namespaced<TAB>kind
#
# IMPORTANT: do not assume fixed column numbers in `kubectl api-resources
# -o wide`. The output is human-readable and its columns have changed over
# time. Read the header and locate NAME, SHORTNAMES, NAMESPACED and KIND by
# name instead. This also makes resources such as Secrets reliably resolve as
# namespaced. Kubernetes documents `api-resources` as the authoritative source
# for whether a resource is namespaced.
_kfz__resource_meta() {
    local wanted=$1
    local table canonical namespaced kind

    # Resolve the canonical resource name and kind from each scope separately.
    # Kubernetes exposes APIResource.namespaced as the authoritative scope.
    # We deliberately do not maintain a hard-coded scope list.
    for namespaced in true false; do
        table=$(
            kubectl api-resources \
                --verbs=list \
                --namespaced="$namespaced" \
                --no-headers \
                2>/dev/null
        ) || continue

        canonical=$(
            awk -v wanted="$wanted" '
                function short_match(value, wanted,    n, a, i) {
                    if (value == "" || value == "<none>") return 0
                    n = split(value, a, ",")
                    for (i = 1; i <= n; i++)
                        if (a[i] == wanted)
                            return 1
                    return 0
                }

                $1 == wanted || short_match($2, wanted) {
                    print $1
                    exit
                }
            ' <<< "$table"
        )

        if [[ -n $canonical ]]; then
            kind=$(
                awk -v wanted="$canonical" '
                    $1 == wanted {
                        print $NF
                        exit
                    }
                ' <<< "$table"
            )

            printf '%s\t%s\t%s\n' "$canonical" "$namespaced" "$kind"
            return 0
        fi
    done

    return 1
}

# Return the current kubeconfig namespace without changing anything.
_kfz__current_namespace() {
    local namespace
    namespace=$(kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)
    printf '%s\n' "${namespace:-default}"
}

# Render the selected Kubernetes object in the preview pane. The command is
# intentionally resource-specific where a better view exists, with describe
# as the generic fallback.
_kfz__preview_resource() {
    local resource=$1
    local namespace=$2
    local name=$3

    [[ -n $name ]] || return 0

    case $resource in
        pods|pod)
            if [[ $namespace == '-' ]]; then
                kubectl describe pod "$name" 2>&1
            else
                kubectl describe pod "$name" -n "$namespace" 2>&1
            fi
            ;;
        deployments|deployment)
            if [[ $namespace == '-' ]]; then
                kubectl describe deployment "$name" 2>&1
            else
                kubectl describe deployment "$name" -n "$namespace" 2>&1
            fi
            ;;
        daemonsets|daemonset)
            if [[ $namespace == '-' ]]; then
                kubectl describe daemonset "$name" 2>&1
            else
                kubectl describe daemonset "$name" -n "$namespace" 2>&1
            fi
            ;;
        statefulsets|statefulset)
            if [[ $namespace == '-' ]]; then
                kubectl describe statefulset "$name" 2>&1
            else
                kubectl describe statefulset "$name" -n "$namespace" 2>&1
            fi
            ;;
        services|service)
            if [[ $namespace == '-' ]]; then
                kubectl describe service "$name" 2>&1
            else
                kubectl describe service "$name" -n "$namespace" 2>&1
            fi
            ;;
        nodes|node)
            kubectl describe node "$name" 2>&1
            ;;
        namespaces|namespace)
            kubectl describe namespace "$name" 2>&1
            ;;
        *)
            if [[ $namespace == '-' ]]; then
                kubectl describe "$resource" "$name" 2>&1
            else
                kubectl describe "$resource" "$name" -n "$namespace" 2>&1
            fi
            ;;
    esac | _kfz__preview_render text
}

# Pick a container for pod exec/attach. stdout is the selected container name.
_kfz__select_container() {
    local resource=$1
    local namespace=$2
    local name=$3
    local -a containers=()
    local container

    [[ $resource == pods || $resource == pod ]] || return 1

    if [[ $namespace == '-' ]]; then
        mapfile -t containers < <(
            kubectl get pod "$name" -o jsonpath='{range .spec.containers[*]}{.name}{"\n"}{end}' 2>/dev/null
        )
    else
        mapfile -t containers < <(
            kubectl get pod "$name" -n "$namespace" -o jsonpath='{range .spec.containers[*]}{.name}{"\n"}{end}' 2>/dev/null
        )
    fi

    ((${#containers[@]})) || return 1

    if ((${#containers[@]} == 1)); then
        printf '%s\n' "${containers[0]}"
        return 0
    fi

    container=$(
        printf '%s\n' "${containers[@]}" |
            fzf \
                --with-shell='bash -c' \
                --exact \
                --no-multi \
                --no-preview \
                --prompt='Container ❯❯❯ ' \
                --header='Enter select · Esc cancel' \
                --height=50% \
                --layout=reverse
    ) || return 1

    printf '%s\n' "$container"
}

# Build resource rows in this stable internal format:
#   namespace<TAB>name<TAB>display-row
#
# The first two fields are private data used by fzf placeholders/actions.
# Field 3 is the ONLY displayed field. It is fully width-padded here, so fzf
# does not need to interpret/reassemble tab-separated display fragments.
_kfz__list() {
    local resource=$1
    local namespaced=$2
    local scope=$3
    local namespace=$4

    if [[ $namespaced == true ]]; then
        if [[ $scope == all ]]; then
            kubectl get "$resource" -A --no-headers -o wide 2>/dev/null |
                awk '
                    {
                        # For -A output: NAMESPACE NAME <rest...>. Capture the
                        # suffix from the original line so kubectl column
                        # spacing remains intact.
                        if (!match($0, /^[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]+/)) next

                        ns = $1
                        name = $2
                        if (ns == "" || name == "") next

                        suffix = substr($0, RSTART + RLENGTH)
                        sub(/^[[:space:]]+/, "", suffix)
                        identity = ns "/" name
                        row_ns[NR] = ns
                        row_name[NR] = name
                        row_identity[NR] = identity
                        row_suffix[NR] = suffix

                        if (length(identity) > max_identity)
                            max_identity = length(identity)
                    }
                    END {
                        for (i = 1; i <= NR; i++)
                            if (row_name[i] != "")
                                printf "%s\t%s\t%-*s  %s\n", \
                                    row_ns[i], row_name[i], max_identity, \
                                    row_identity[i], row_suffix[i]
                    }
                '
        else
            kubectl get "$resource" -n "$namespace" --no-headers -o wide 2>/dev/null |
                awk -v ns="$namespace" '
                    {
                        # For a namespaced, single-namespace kubectl get, the
                        # first whitespace-delimited field is NAME.  The old
                        # formatter used RSTART (always 1 for this regex) as a
                        # substring length, which reduced every name to its
                        # first character.
                        name = $1
                        if (name == "") next

                        # Keep kubectl complete column suffix verbatim.
                        match($0, /^[^[:space:]]+/)
                        suffix = substr($0, RSTART + RLENGTH)
                        sub(/^[[:space:]]+/, "", suffix)
                        identity = ns "/" name
                        row_name[NR] = name
                        row_identity[NR] = identity
                        row_suffix[NR] = suffix

                        if (length(identity) > max_identity)
                            max_identity = length(identity)
                    }
                    END {
                        for (i = 1; i <= NR; i++)
                            if (row_name[i] != "")
                                printf "%s\t%s\t%-*s  %s\n", \
                                    ns, row_name[i], max_identity, \
                                    row_identity[i], row_suffix[i]
                    }
                '
        fi
    else
        kubectl get "$resource" --no-headers -o wide 2>/dev/null |
            awk '
                {
                    # For cluster-scoped resources the first field is NAME.
                    # Preserve the complete suffix emitted by kubectl.
                    name = $1
                    if (name == "") next

                    match($0, /^[^[:space:]]+/)
                    suffix = substr($0, RSTART + RLENGTH)
                    sub(/^[[:space:]]+/, "", suffix)
                    row_name[NR] = name
                    row_identity[NR] = name
                    row_suffix[NR] = suffix

                    if (length(name) > max_identity)
                        max_identity = length(name)
                }
                END {
                    for (i = 1; i <= NR; i++)
                        if (row_name[i] != "")
                            printf "-\t%s\t%-*s  %s\n", \
                                row_name[i], max_identity, \
                                row_identity[i], row_suffix[i]
                }
            '
    fi
}

# Reload using the namespace/scope state in FZF_PROMPT.
_kfz__reload() {
    local resource=$1
    local namespaced=$2
    local namespace=$3

    if [[ ${FZF_PROMPT-} == *'[ALL]'* ]]; then
        _kfz__list "$resource" "$namespaced" all "$namespace"
    else
        _kfz__list "$resource" "$namespaced" current "$namespace"
    fi
}

# Toggle current namespace <-> all namespaces. The transform action emits a
# new pair of fzf actions, allowing the stateful prompt and reload source to be
# changed without restarting fzf.
_kfz__toggle_scope() {
    local resource=$1
    local namespaced=$2
    local namespace=$3
    local prompt

    if [[ ${FZF_PROMPT-} == *'[ALL]'* ]]; then
        prompt="K8s:$resource [ns:$namespace] ❯❯❯ "
        printf 'change-prompt(%s)+reload(_kfz__list %q %q current %q)' \
            "$prompt" "$resource" "$namespaced" "$namespace"
    else
        prompt="K8s:$resource [ALL] ❯❯❯ "
        printf 'change-prompt(%s)+reload(_kfz__list %q %q all %q)' \
            "$prompt" "$resource" "$namespaced" "$namespace"
    fi
}

# A compact two-line key legend. Keeping it short enough for the normal fzf
# width prevents the header itself from becoming a one-line truncated blob.
_kfz__resource_header() {
    local resource=$1

    printf '%s\n%s' \
        'Enter describe · Tab select · Ctrl-D delete · Ctrl-E edit · Ctrl-Y YAML' \
        'Ctrl-R reload · Ctrl-A all-ns · Alt-C name · Alt-N namespace · Ctrl-/ preview'

    case $resource in
        pods|pod)
            printf '\n%s\n%s' \
                'Pod: Ctrl-L logs · Alt-L follow · Alt-P previous · Ctrl-X exec · Alt-A attach' \
                'Pod: Alt-F forward · Ctrl-T top'
            ;;
        deployments|deployment|daemonsets|daemonset|statefulsets|statefulset)
            printf '\n%s\n%s' \
                'Workload: Alt-R restart · Alt-U undo · Alt-H history · Alt-W status' \
                'Workload: Alt-S scale · Alt-F forward'
            ;;
        services|service)
            printf '\n%s' 'Service: Alt-F port-forward'
            ;;
        nodes|node)
            printf '\n%s' 'Nodes: Ctrl-T top · Alt-U cordon/uncordon · Alt-R drain'
            ;;
        replicasets|replicaset)
            printf '\n%s' 'ReplicaSets: Alt-S scale'
            ;;
    esac
}

# -----------------------------------------------------------------------------
# Kubernetes actions
# -----------------------------------------------------------------------------

# Read fzf's {+f} file. Every line is:
#   namespace<TAB>name<TAB>display-row
_kfz__action() {
    local action=$1
    local resource=$2

    [[ ${3-} == --file ]] || {
        printf 'kfz: internal error: expected --file <fzf temp file>\n' >&2
        return 2
    }

    local file=${4-}
    [[ -r $file ]] || {
        printf 'kfz: selected-resource file is not readable\n' >&2
        return 2
    }

    local -a ns_list=() name_list=()
    local ns name rest line identity

    while IFS= read -r line; do
        [[ -n $line ]] || continue

        # Normally {+f} contains the original tab-delimited row. If a future
        # fzf presentation transform supplies the displayed row instead, the
        # fallback below recovers namespace/name from its identity prefix.
        IFS=$'\t' read -r ns name rest <<< "$line"
        if [[ -z $name ]]; then
            identity=${line%%[[:space:]]*}
            if [[ $identity == */* ]]; then
                ns=${identity%%/*}
                name=${identity#*/}
            else
                ns='-'
                name=$identity
            fi
        fi

        [[ -n $name ]] || continue
        ns_list+=("$ns")
        name_list+=("$name")
    done < "$file"

    ((${#name_list[@]})) || return 0

    _kfz__kubectl_target() {
        local target_resource=$1 target_ns=$2 target_name=$3
        if [[ $target_ns == '-' || -z $target_ns ]]; then
            printf '%s/%s' "$target_resource" "$target_name"
        else
            printf '%s/%s -n %q' "$target_resource" "$target_name" "$target_ns"
        fi
    }

    _kfz__run_describe() {
        local i
        for i in "${!name_list[@]}"; do
            printf '\n===== %s =====\n' \
                "$(_kfz__kubectl_target "$resource" "${ns_list[i]}" "${name_list[i]}")"
            if [[ ${ns_list[i]} == '-' ]]; then
                kubectl describe "$resource" "${name_list[i]}"
            else
                kubectl describe "$resource" "${name_list[i]}" -n "${ns_list[i]}"
            fi
        done
    }

    case $action in
        describe)
            local page_file page_rc
            page_file=$(mktemp "${TMPDIR:-/tmp}/kfz-describe.XXXXXX") || return 1

            _kfz__run_describe >"$page_file" 2>&1
            page_rc=$?

            _kfz__page_file text "$page_file"
            local pager_rc=$?

            rm -f -- "$page_file"

            # Prefer the kubectl/describe status, but report a pager failure if
            # kubectl itself succeeded and the pager did not.
            if ((page_rc != 0)); then
                return "$page_rc"
            fi
            return "$pager_rc"
            ;;

        yaml)
            local page_file page_rc=0 kubectl_rc pager_rc i
            page_file=$(mktemp "${TMPDIR:-/tmp}/kfz-yaml.XXXXXX") || return 1

            : >"$page_file"

            for i in "${!name_list[@]}"; do
                printf '\n===== %s =====\n' \
                    "$(_kfz__kubectl_target "$resource" "${ns_list[i]}" "${name_list[i]}")" \
                    >>"$page_file"

                if [[ ${ns_list[i]} == '-' ]]; then
                    kubectl get "$resource" "${name_list[i]}" -o yaml >>"$page_file" 2>&1
                else
                    kubectl get "$resource" "${name_list[i]}" -n "${ns_list[i]}" -o yaml >>"$page_file" 2>&1
                fi
                kubectl_rc=$?

                # Keep collecting selected resources even if one lookup fails,
                # while preserving the first kubectl error for the final status.
                if ((kubectl_rc != 0 && page_rc == 0)); then
                    page_rc=$kubectl_rc
                fi
            done

            _kfz__page_file yaml "$page_file"
            pager_rc=$?

            rm -f -- "$page_file"

            if ((page_rc != 0)); then
                return "$page_rc"
            fi
            return "$pager_rc"
            ;;

        edit)
            local i
            for i in "${!name_list[@]}"; do
                if [[ ${ns_list[i]} == '-' ]]; then
                    kubectl edit "$resource" "${name_list[i]}"
                else
                    kubectl edit "$resource" "${name_list[i]}" -n "${ns_list[i]}"
                fi
            done
            ;;

        delete)
            printf 'Selected resources:\n'
            local i
            for i in "${!name_list[@]}"; do
                printf '  %s\n' "$(_kfz__kubectl_target "$resource" "${ns_list[i]}" "${name_list[i]}")"
            done
            printf '\n'

            local answer
            read -r -p "Delete ${#name_list[@]} resource(s)? [y/N] " answer </dev/tty
            [[ $answer =~ ^[Yy]$ ]] || return 0

            for i in "${!name_list[@]}"; do
                if [[ ${ns_list[i]} == '-' ]]; then
                    kubectl delete "$resource" "${name_list[i]}"
                else
                    kubectl delete "$resource" "${name_list[i]}" -n "${ns_list[i]}"
                fi
            done
            ;;

        copy-name)
            local output='' i
            for i in "${!name_list[@]}"; do
                output+="${name_list[i]}"$'\n'
            done
            printf '%s' "$output" | _kfz__clip
            ;;

        copy-namespace)
            local output='' i
            for i in "${!ns_list[@]}"; do
                [[ ${ns_list[i]} != '-' ]] || continue
                output+="${ns_list[i]}"$'\n'
            done
            printf '%s' "$output" | _kfz__clip
            ;;

        logs)
            [[ $resource == pods || $resource == pod ]] || {
                printf 'kfz: logs are only available from a pod picker.\n' >&2
                _kfz__pause
                return 1
            }
            local i
            for i in "${!name_list[@]}"; do
                printf '\n===== %s =====\n' \
                    "$(_kfz__kubectl_target pod "${ns_list[i]}" "${name_list[i]}")"
                if [[ ${ns_list[i]} == '-' ]]; then
                    kubectl logs "${name_list[i]}" --all-containers --prefix --tail=200
                else
                    kubectl logs "${name_list[i]}" -n "${ns_list[i]}" --all-containers --prefix --tail=200
                fi
            done
            _kfz__pause
            ;;

        logs-follow)
            [[ $resource == pods || $resource == pod ]] || {
                printf 'kfz: logs are only available from a pod picker.\n' >&2
                _kfz__pause
                return 1
            }
            ((${#name_list[@]} == 1)) || {
                printf 'kfz: follow requires exactly one selected pod.\n' >&2
                _kfz__pause
                return 1
            }
            if [[ ${ns_list[0]} == '-' ]]; then
                kubectl logs -f "${name_list[0]}" --all-containers --prefix --tail=200
            else
                kubectl logs -f "${name_list[0]}" -n "${ns_list[0]}" --all-containers --prefix --tail=200
            fi
            ;;

        logs-previous)
            [[ $resource == pods || $resource == pod ]] || {
                printf 'kfz: logs are only available from a pod picker.\n' >&2
                _kfz__pause
                return 1
            }
            local i
            for i in "${!name_list[@]}"; do
                printf '\n===== previous: %s =====\n' \
                    "$(_kfz__kubectl_target pod "${ns_list[i]}" "${name_list[i]}")"
                if [[ ${ns_list[i]} == '-' ]]; then
                    kubectl logs "${name_list[i]}" --all-containers --prefix --previous
                else
                    kubectl logs "${name_list[i]}" -n "${ns_list[i]}" --all-containers --prefix --previous
                fi
            done
            _kfz__pause
            ;;

        exec|attach)
            [[ $resource == pods || $resource == pod ]] || {
                printf 'kfz: %s is only available from a pod picker.\n' "$action" >&2
                _kfz__pause
                return 1
            }
            ((${#name_list[@]} == 1)) || {
                printf 'kfz: %s requires exactly one selected pod.\n' "$action" >&2
                _kfz__pause
                return 1
            }

            local container
            container=$(_kfz__select_container pod "${ns_list[0]}" "${name_list[0]}") || return 1

            if [[ $action == exec ]]; then
                if [[ ${ns_list[0]} == '-' ]]; then
                    if kubectl exec "${name_list[0]}" -c "$container" -- test -x /bin/bash >/dev/null 2>&1; then
                        kubectl exec -it "${name_list[0]}" -c "$container" -- /bin/bash -il
                    else
                        kubectl exec -it "${name_list[0]}" -c "$container" -- /bin/sh
                    fi
                else
                    if kubectl exec "${name_list[0]}" -n "${ns_list[0]}" -c "$container" -- test -x /bin/bash >/dev/null 2>&1; then
                        kubectl exec -it "${name_list[0]}" -n "${ns_list[0]}" -c "$container" -- /bin/bash -il
                    else
                        kubectl exec -it "${name_list[0]}" -n "${ns_list[0]}" -c "$container" -- /bin/sh
                    fi
                fi
            elif [[ ${ns_list[0]} == '-' ]]; then
                kubectl attach -it "${name_list[0]}" -c "$container"
            else
                kubectl attach -it "${name_list[0]}" -n "${ns_list[0]}" -c "$container"
            fi
            ;;

        port-forward)
            case $resource in
                pods|pod|services|service|deployments|deployment) ;;
                *)
                    printf 'kfz: port-forward is enabled for pods, services and deployments.\n' >&2
                    _kfz__pause
                    return 1
                    ;;
            esac
            ((${#name_list[@]} == 1)) || {
                printf 'kfz: port-forward requires exactly one selected resource.\n' >&2
                _kfz__pause
                return 1
            }
            local port_spec
            printf 'Examples: 8080:80, 8443:https, :5000\n'
            read -r -p 'Port forward [LOCAL:REMOTE]: ' port_spec </dev/tty
            [[ -n $port_spec ]] || return 0
            [[ $port_spec =~ ^[0-9]+:[A-Za-z0-9._-]+$ ||
               $port_spec =~ ^:[A-Za-z0-9._-]+$ ||
               $port_spec =~ ^[0-9]+$ ]] || {
                printf 'kfz: invalid port specification: %s\n' "$port_spec" >&2
                _kfz__pause
                return 1
            }
            local target="${resource}/${name_list[0]}"
            if [[ ${ns_list[0]} == '-' ]]; then
                kubectl port-forward "$target" "$port_spec"
            else
                kubectl port-forward "$target" -n "${ns_list[0]}" "$port_spec"
            fi
            ;;

        restart|undo|rollout-history|rollout-status)
            case $resource in
                deployments|deployment|daemonsets|daemonset|statefulsets|statefulset) ;;
                *)
                    printf 'kfz: rollout actions are only available for deployments, daemonsets and statefulsets.\n' >&2
                    _kfz__pause
                    return 1
                    ;;
            esac
            local rollout_action
            case $action in
                restart) rollout_action=restart ;;
                undo) rollout_action=undo ;;
                rollout-history) rollout_action=history ;;
                rollout-status) rollout_action=status ;;
            esac
            local i
            for i in "${!name_list[@]}"; do
                printf '\n===== %s =====\n' \
                    "$(_kfz__kubectl_target "$resource" "${ns_list[i]}" "${name_list[i]}")" >&2
                if [[ ${ns_list[i]} == '-' ]]; then
                    kubectl rollout "$rollout_action" "$resource" "${name_list[i]}"
                else
                    kubectl rollout "$rollout_action" "$resource" "${name_list[i]}" -n "${ns_list[i]}"
                fi
            done
            [[ $action == rollout-history ]] && _kfz__pause
            ;;

        scale)
            case $resource in
                deployments|deployment|daemonsets|daemonset|statefulsets|statefulset|replicasets|replicaset) ;;
                *)
                    printf 'kfz: scale is enabled for deployments, daemonsets, statefulsets and replicasets.\n' >&2
                    _kfz__pause
                    return 1
                    ;;
            esac
            local replicas
            read -r -p 'Desired replicas: ' replicas </dev/tty
            [[ $replicas =~ ^[0-9]+$ ]] || {
                printf 'kfz: replicas must be a non-negative integer.\n' >&2
                _kfz__pause
                return 1
            }
            local i
            for i in "${!name_list[@]}"; do
                if [[ ${ns_list[i]} == '-' ]]; then
                    kubectl scale "$resource" "${name_list[i]}" --replicas="$replicas"
                else
                    kubectl scale "$resource" "${name_list[i]}" -n "${ns_list[i]}" --replicas="$replicas"
                fi
            done
            ;;

        top)
            case $resource in
                pods|pod)
                    local i
                    for i in "${!name_list[@]}"; do
                        if [[ ${ns_list[i]} == '-' ]]; then
                            kubectl top pod "${name_list[i]}"
                        else
                            kubectl top pod "${name_list[i]}" -n "${ns_list[i]}"
                        fi
                    done
                    ;;
                nodes|node)
                    local i
                    for i in "${!name_list[@]}"; do
                        kubectl top node "${name_list[i]}"
                    done
                    ;;
                *)
                    printf 'kfz: top is enabled for pods and nodes.\n' >&2
                    _kfz__pause
                    return 1
                    ;;
            esac
            _kfz__pause
            ;;

        events)
            for i in "${!name_list[@]}"; do
                if [[ ${ns_list[i]} == '-' ]]; then
                    kubectl get events --field-selector "involvedObject.name=${name_list[i]}" --sort-by=.lastTimestamp 
                else
                    kubectl get events -n "${ns_list[i]}" --field-selector "involvedObject.name=${name_list[i]}" --sort-by=.lastTimestamp 
                fi
            done
            _kfz__pause
            ;;
        cordon-toggle)
            [[ $resource == nodes || $resource == node ]] || {
                printf 'kfz: cordon/uncordon is only available from a node picker.\n' >&2
                _kfz__pause
                return 1
            }
            local i unschedulable
            for i in "${!name_list[@]}"; do
                unschedulable=$(kubectl get node "${name_list[i]}" -o jsonpath='{.spec.unschedulable}' 2>/dev/null)
                if [[ $unschedulable == true ]]; then
                    kubectl uncordon "${name_list[i]}"
                else
                    kubectl cordon "${name_list[i]}"
                fi
            done
            ;;

        drain)
            [[ $resource == nodes || $resource == node ]] || {
                printf 'kfz: drain is only available from a node picker.\n' >&2
                _kfz__pause
                return 1
            }
            local i answer
            printf 'Selected node(s):\n'
            for i in "${!name_list[@]}"; do
                printf '  %s\n' "${name_list[i]}"
            done
            printf '\nDrain will evict eligible workloads and may delete emptyDir data.\n'
            read -r -p 'Proceed with drain? [y/N] ' answer </dev/tty
            [[ $answer =~ ^[Yy]$ ]] || return 0
            for i in "${!name_list[@]}"; do
                kubectl drain "${name_list[i]}" --ignore-daemonsets --delete-emptydir-data
            done
            ;;

        *)
            printf 'kfz: unknown action: %s\n' "$action" >&2
            return 2
            ;;
    esac
}

# -----------------------------------------------------------------------------
# Resource picker
# -----------------------------------------------------------------------------

_kfz__resource_picker() {
    local requested=${1:-}
    shift || true

    local namespace=''
    local -a query_parts=()
    local arg

    # Parse -n/--namespace anywhere after the resource name.
    while (($#)); do
        arg=$1
        case $arg in
            -n|--namespace)
                if (($# < 2)); then
                    printf 'kfz: %s requires a namespace.\n' "$arg" >&2
                    return 2
                fi
                namespace=$2
                shift 2
                ;;
            --)
                shift
                query_parts+=("$@")
                break
                ;;
            *)
                query_parts+=("$arg")
                shift
                ;;
        esac
    done

    local initial_query=''
    ((${#query_parts[@]})) && initial_query=${query_parts[*]}

    if [[ -z $requested ]]; then
        _kfz__resource_types "$namespace" "$initial_query"
        return $?
    fi

    local current_namespace
    current_namespace=$(_kfz__current_namespace)
    [[ -n $namespace ]] || namespace=$current_namespace

    local meta canonical namespaced kind
    meta=$(_kfz__resource_meta "$requested") || true
    if [[ -z $meta ]]; then
        printf 'kfz: unknown/list-incompatible resource: %s\n' "$requested" >&2
        printf 'Use `kubectl api-resources` or `kfz` to choose a resource type.\n' >&2
        return 2
    fi

    IFS=$'\t' read -r canonical namespaced kind <<< "$meta"
    [[ -n $canonical ]] || return 2

    local prompt
    if [[ $namespaced != true ]]; then
        prompt="K8s:$canonical [cluster] ❯❯❯ "
    else
        prompt="K8s:$canonical [ns:$namespace] ❯❯❯ "
    fi

    # Keep the user-facing row in field 3. Fields 1/2 remain hidden but are
    # available to fzf placeholders for actions and previews.
    local preview_cmd
    if [[ $namespaced == true ]]; then
        printf -v preview_cmd 'kubectl describe %q {2} -n {1} 2>&1 | _kfz__preview_render text' "$canonical"
    else
        printf -v preview_cmd 'kubectl describe %q {2} 2>&1 | _kfz__preview_render text' "$canonical"
    fi

    local reload_current reload_all
    printf -v reload_current '_kfz__list %q %q current %q' "$canonical" "$namespaced" "$namespace"
    printf -v reload_all '_kfz__list %q %q all %q' "$canonical" "$namespaced" "$namespace"

    local accept_nth
    if [[ $namespaced == true ]]; then
        accept_nth='{1}/{2}'
    else
        accept_nth="${canonical}/{2}"
    fi

    local -a args=(
        --with-shell='bash -c'
        --exact
        --multi
        --delimiter=$'\t'
        --with-nth=3
        --accept-nth="$accept_nth"
        --prompt="$prompt"
        --query="$initial_query"
        --header="$(_kfz__resource_header "$canonical")"
        --preview="$preview_cmd"
        --preview-window='right:55%:hidden:wrap'
        --bind="ctrl-r:reload($reload_current)"
        --bind="enter:execute(_kfz__action describe '$canonical' --file {+f})"
        --bind="ctrl-d:execute(_kfz__action delete '$canonical' --file {+f})+reload($reload_current)"
        --bind="ctrl-e:execute(_kfz__action edit '$canonical' --file {+f})+reload($reload_current)"
        --bind="ctrl-y:execute(_kfz__action yaml '$canonical' --file {+f})"
        --bind="alt-c:execute-silent(_kfz__action copy-name '$canonical' --file {+f})"
        --bind="alt-n:execute-silent(_kfz__action copy-namespace '$canonical' --file {+f})"
        --bind='alt-enter:accept'
    )

    if [[ $namespaced == true ]]; then
        args+=(
            --bind="ctrl-a:transform:_kfz__toggle_scope '$canonical' '$namespaced' '$namespace'"
        )
    fi

    case $canonical in
        pods|pod)
            args+=(
                --bind="ctrl-l:execute(_kfz__action logs '$canonical' --file {+f})"
                --bind="alt-l:execute(_kfz__action logs-follow '$canonical' --file {+f})"
                --bind="alt-p:execute(_kfz__action logs-previous '$canonical' --file {+f})"
                --bind="ctrl-x:execute(_kfz__action exec '$canonical' --file {+f})"
                --bind="alt-a:execute(_kfz__action attach '$canonical' --file {+f})"
                --bind="alt-f:execute(_kfz__action port-forward '$canonical' --file {+f})"
                --bind="ctrl-t:execute(_kfz__action top '$canonical' --file {+f})"
                --bind="alt-e:execute(_kfz__action events '$canonical' --file {+f})"
            )
            ;;
        deployments|deployment|daemonsets|daemonset|statefulsets|statefulset)
            args+=(
                --bind="alt-f:execute(_kfz__action port-forward '$canonical' --file {+f})"
                --bind="alt-r:execute(_kfz__action restart '$canonical' --file {+f})+reload($reload_current)"
                --bind="alt-u:execute(_kfz__action undo '$canonical' --file {+f})+reload($reload_current)"
                --bind="alt-h:execute(_kfz__action rollout-history '$canonical' --file {+f})"
                --bind="alt-w:execute(_kfz__action rollout-status '$canonical' --file {+f})"
                --bind="alt-s:execute(_kfz__action scale '$canonical' --file {+f})+reload($reload_current)"
            )
            ;;
        services|service)
            args+=(
                --bind="alt-f:execute(_kfz__action port-forward '$canonical' --file {+f})"
            )
            ;;
        nodes|node)
            args+=(
                --bind="ctrl-t:execute(_kfz__action top '$canonical' --file {+f})"
                --bind="alt-u:execute(_kfz__action cordon-toggle '$canonical' --file {+f})+reload($reload_current)"
                --bind="alt-r:execute(_kfz__action drain '$canonical' --file {+f})+reload($reload_current)"
            )
            ;;
        replicasets|replicaset)
            args+=(
                --bind="alt-s:execute(_kfz__action scale '$canonical' --file {+f})+reload($reload_current)"
            )
            ;;
    esac

    _kfz__list "$canonical" "$namespaced" current "$namespace" |
        fzf "${args[@]}"
}

# -----------------------------------------------------------------------------
# Resource-type chooser (bare `kfz` and the namespace gate use this)
# -----------------------------------------------------------------------------

# Internal format:
#   resource<TAB>shortnames<TAB>namespaced<TAB>kind<TAB>display
_kfz__resource_types_list() {
    {
        kubectl api-resources --verbs=list --namespaced=true 2>/dev/null
        kubectl api-resources --verbs=list --namespaced=false 2>/dev/null
    } |
        awk '
            NR == 1 {
                name_col = short_col = ns_col = kind_col = 0
                for (i = 1; i <= NF; i++) {
                    if ($i == "NAME") name_col = i
                    else if ($i == "SHORTNAMES") short_col = i
                    else if ($i == "NAMESPACED") ns_col = i
                    else if ($i == "KIND") kind_col = i
                }
                next
            }

            # The second api-resources invocation has its own header.
            $1 == "NAME" && $2 == "SHORTNAMES" {
                next
            }

            name_col && ns_col && kind_col && NF >= kind_col {
                resource = $name_col
                short = (short_col && $short_col != "<none>" ? $short_col : "-")
                namespaced = $ns_col
                kind = $kind_col

                key = resource SUBSEP namespaced
                if (seen[key]++)
                    next

                row_resource[++count] = resource
                row_short[count] = short
                row_namespaced[count] = namespaced
                row_kind[count] = kind

                if (length(resource) > w1) w1 = length(resource)
                if (length(short) > w2) w2 = length(short)
                if (length(namespaced) > w3) w3 = length(namespaced)
            }

            END {
                for (i = 1; i <= count; i++)
                    printf "%s\t%s\t%s\t%s\t%-*s  %-*s  %-*s  %s\n", \
                        row_resource[i], row_short[i], row_namespaced[i], \
                        row_kind[i], w1, row_resource[i], w2, row_short[i], \
                        w3, row_namespaced[i], row_kind[i]
            }
        '
}

_kfz__resource_type_header() {
    printf '%s\n%s' \
        'Enter browse · Ctrl-R reload · Alt-C copy name · Ctrl-/ preview' \
        'Exact matching · Esc exit'
}

_kfz__resource_types() {
    local namespace=${1-}
    local initial_query=${2-}
    local selected resource
    local preview_cmd

    _kfz__require_commands || return

    if [[ -n $namespace ]]; then
        printf -v preview_cmd 'kubectl explain {1} 2>&1 | _kfz__preview_render text'
    else
        printf -v preview_cmd 'kubectl explain {1} 2>&1 | _kfz__preview_render text'
    fi

    selected=$(
        _kfz__resource_types_list |
            fzf \
                --with-shell='bash -c' \
                --exact \
                --no-multi \
                --delimiter=$'\t' \
                --with-nth=5 \
                --accept-nth='{1}' \
                --prompt="K8s resource${namespace:+ [ns:$namespace]} ❯❯❯ " \
                --query="$initial_query" \
                --header="$(_kfz__resource_type_header)" \
                --preview="$preview_cmd" \
                --preview-window='right:55%:hidden:wrap' \
                --bind='ctrl-r:reload(_kfz__resource_types_list)' \
                --bind='alt-c:execute-silent(printf %s {1} | _kfz__clip)' \
                --bind='enter:accept'
    ) || return 0

    resource=$selected
    [[ -n $resource ]] || return 0

    if [[ -n $namespace ]]; then
        _kfz__resource_picker "$resource" -n "$namespace"
    else
        _kfz__resource_picker "$resource"
    fi
}

# -----------------------------------------------------------------------------
# Namespace gate
# -----------------------------------------------------------------------------

# This deliberately does NOT call `kubectl config set-context --current ...`.
# Enter means: "use this namespace as the starting scope for resource
# exploration". The selected namespace is passed into the resource-type chooser.
_kfz__namespace_gate_action() {
    local namespace=$1

    [[ -n $namespace ]] || return 0
    _kfz__resource_types "$namespace"
}

# Internal format:
#   namespace<TAB>status<TAB>display
_kfz__namespaces_list() {
    local current
    current=$(_kfz__current_namespace)

    kubectl get namespaces --no-headers \
        -o custom-columns='NAME:.metadata.name,STATUS:.status.phase' 2>/dev/null |
        awk -v current="$current" '
            {
                name = $1
                status = ($2 == "" ? "-" : $2)
                marker = (name == current ? " *" : "")

                row_name[NR] = name
                row_status[NR] = status marker
                if (length(name) > w1) w1 = length(name)
                if (length(status marker) > w2) w2 = length(status marker)
            }
            END {
                for (i = 1; i <= NR; i++)
                    if (row_name[i] != "")
                        printf "%s\t%s\t%-*s  %-*s\n", \
                            row_name[i], row_status[i], w1, row_name[i], \
                            w2, row_status[i]
            }
        '
}

_kfz__namespace_header() {
    printf '%s\n%s' \
        'Enter explore resources · Ctrl-R reload · Ctrl-D delete · Alt-C copy' \
        'Current namespace is marked with * · Ctrl-/ preview · Esc exit'
}

_kfz__namespace_action() {
    local action=$1
    local namespace=$2

    case $action in
        delete)
            printf 'Namespace: %s\n' "$namespace"
            printf 'This deletes the namespace and resources governed by its lifecycle.\n\n'
            local answer confirm
            read -r -p 'Delete namespace? Type the namespace name exactly: ' confirm </dev/tty
            [[ $confirm == "$namespace" ]] || return 0
            read -r -p 'Final confirmation [y/N]: ' answer </dev/tty
            [[ $answer =~ ^[Yy]$ ]] || return 0
            kubectl delete namespace "$namespace"
            ;;
        *)
            printf 'kfz: unknown namespace action: %s\n' "$action" >&2
            return 2
            ;;
    esac
}

_kfz__namespaces() {
    local -a query_parts=()
    local arg initial_query=''

    _kfz__require_commands || return

    while (($#)); do
        arg=$1
        case $arg in
            -n|--namespace)
                printf 'kfz: -n/--namespace is not applicable to the namespace gate.\n' >&2
                return 2
                ;;
            --)
                shift
                query_parts+=("$@")
                break
                ;;
            *)
                query_parts+=("$arg")
                shift
                ;;
        esac
    done
    ((${#query_parts[@]})) && initial_query=${query_parts[*]}

    # Enter executes the namespace gate in-place. There is deliberately no
    # accepted namespace value here because this picker is an explorer gate,
    # not a kubeconfig namespace switcher.
    _kfz__namespaces_list |
        fzf \
            --with-shell='bash -c' \
            --exact \
            --no-multi \
            --delimiter=$'\t' \
            --with-nth=3 \
            --accept-nth='{1}' \
            --prompt='K8s namespace ❯❯❯ ' \
            --query="$initial_query" \
            --header="$(_kfz__namespace_header)" \
            --preview='kubectl describe namespace {1} 2>&1 | _kfz__preview_render text' \
            --preview-window='right:55%:hidden:wrap' \
            --bind='ctrl-r:reload(_kfz__namespaces_list)' \
            --bind='enter:execute(_kfz__namespace_gate_action {1})' \
            --bind='ctrl-d:execute(_kfz__namespace_action delete {1})+reload(_kfz__namespaces_list)' \
            --bind='alt-c:execute-silent(printf %s {1} | _kfz__clip)'
}


# -----------------------------------------------------------------------------
# Context picker
# -----------------------------------------------------------------------------

_kfz__contexts_list() {
    local current
    current=$(kubectl config current-context 2>/dev/null) || current=''

    kubectl config get-contexts -o name 2>/dev/null |
       awk -v current="$current" '{ marker = ($0 == current) ? "*" : " "; printf "%s\t%s\t%s  %s\n", $0, marker, marker, $0 }'
}

_kfz__context_header() {
    printf '%s\n%s' \
        'Enter switch · Ctrl-R reload · Ctrl-D delete · Alt-C copy' \
        'Exact matching · Ctrl-/ preview · Esc exit'
}

_kfz__context_action() {
    local action=$1
    local context=$2

    case $action in
        use)
            kubectl config use-context "$context"
            ;;
        delete)
            local answer
            printf 'Context: %s\n' "$context"
            read -r -p 'Delete this kubeconfig context? [y/N] ' answer </dev/tty
            [[ $answer =~ ^[Yy]$ ]] || return 0
            kubectl config delete-context "$context"
            ;;
        *)
            printf 'kfz: unknown context action: %s\n' "$action" >&2
            return 2
            ;;
    esac
}

_kfz__context_preview() {
    local context=$1

    kubectl config get-contexts "$context" 2>&1 |
        _kfz__preview_render text
}

_kfz__contexts() {
    _kfz__require_commands || return

    local selected
    selected=$(
        _kfz__contexts_list |
            fzf \
                --with-shell='bash -c' \
                --exact \
                --no-multi \
                --delimiter=$'\t' \
                --with-nth=3 \
                --accept-nth='{1}' \
                --prompt='K8s context ❯❯❯ ' \
                --header="$(_kfz__context_header)" \
                --preview='kubectl config get-contexts {1} 2>&1 | _kfz__preview_render text' \
                --preview-window='right:55%:hidden:wrap' \
                --bind='ctrl-r:reload(_kfz__contexts_list)' \
                --bind='enter:execute(_kfz__context_action use {1})+reload(_kfz__contexts_list)' \
                --bind='ctrl-d:execute(_kfz__context_action delete {1})+reload(_kfz__contexts_list)' \
                --bind='alt-c:execute-silent(printf %s {1} | _kfz__clip)'
    ) || return 0

    [[ -n $selected ]] && return 0
}

# -----------------------------------------------------------------------------
# Public entry point
# -----------------------------------------------------------------------------

_kfz__help() {
    cat <<'EOF'
kfz - fzf-driven kubectl/K9s-style workflows

Usage:
  kfz                         Choose a resource type, then browse it.
  kfz pods                    Browse pods in the current namespace.
  kfz pods -n observability   Browse pods in a specific namespace.
  kfz -n observability pods   Same, with namespace before the resource.
  kfz deploy                  Browse deployments.
  kfz ns                      Choose a namespace, then choose a resource type.
  kfz resource -n NAME        Choose a resource type scoped to NAME.
  kfz ctx                     Choose/switch kubeconfig context.

Namespace selection:
  kfz ns                      Namespace gate; NEVER changes kubeconfig namespace.
  kfz -n NAME RESOURCE        Start a namespaced resource picker in NAME.
  kfz RESOURCE -n NAME        Same, with -n after the resource.

Resource picker keys:
  Enter       describe selected resource(s)
  Alt-Enter   accept selected resource(s) as namespace/name
  Tab         select multiple
  Ctrl-R      reload
  Ctrl-A      toggle current namespace / all namespaces
  Ctrl-D      delete (confirmation)
  Ctrl-E      edit
  Ctrl-Y      YAML
  Alt-C       copy resource name(s)
  Alt-N       copy namespace(s)
  Ctrl-/      toggle preview (preview starts hidden)

Pod keys:
  Ctrl-L      logs (last 200 lines, all containers)
  Alt-L       follow logs (one pod only)
  Alt-P       previous logs
  Ctrl-X      exec (container picker when needed)
  Alt-A       attach (container picker when needed)
  Alt-F       port-forward
  Ctrl-T      kubectl top pod

Workload keys (Deployment/DaemonSet/StatefulSet):
  Alt-R       rollout restart
  Alt-U       rollout undo
  Alt-H       rollout history
  Alt-W       rollout status
  Alt-S       scale
  Alt-F       port-forward

Node keys:
  Ctrl-T      kubectl top node
  Alt-U       cordon / uncordon
  Alt-R       drain (confirmation)

Matching and preview:
  All kfz fzf pickers use --exact by default.
  Kubernetes previews are hidden initially and use Ctrl-/ to opt in.
  Enter/Describe uses bat with paging when available, then less -R, then cat.
  Preview output remains non-paged so it stays usable inside fzf.
EOF
}

kfz() {
    _kfz__require_commands || return

    local namespace=''

    # Support the namespace before the resource: `kfz -n foo pods`.
    if [[ ${1-} == -n || ${1-} == --namespace ]]; then
        if (($# < 2)); then
            printf 'kfz: %s requires a namespace.\n' "$1" >&2
            return 2
        fi
        namespace=$2
        shift 2

        if [[ ${1-} == ns || ${1-} == namespace || ${1-} == namespaces ]]; then
            printf 'kfz: -n/--namespace cannot be used with the namespace gate.\n' >&2
            return 2
        fi

        case ${1-} in
            resource|resources|type|types)
                shift
                _kfz__resource_types "$namespace" "$*"
                ;;
            '')
                _kfz__resource_types "$namespace"
                ;;
            *)
                _kfz__resource_picker "$1" -n "$namespace" "${@:2}"
                ;;
        esac
        return $?
    fi

    case ${1-} in
        ctx|context|contexts)
            shift
            _kfz__contexts "$@"
            ;;
        ns|namespace|namespaces)
            shift
            _kfz__namespaces "$@"
            ;;
        resource|resources|type|types)
            shift
            local resource_namespace=''
            local -a resource_query_parts=()
            local arg
            while (($#)); do
                arg=$1
                case $arg in
                    -n|--namespace)
                        if (($# < 2)); then
                            printf 'kfz: %s requires a namespace.\n' "$arg" >&2
                            return 2
                        fi
                        resource_namespace=$2
                        shift 2
                        ;;
                    --)
                        shift
                        resource_query_parts+=("$@")
                        break
                        ;;
                    *)
                        resource_query_parts+=("$arg")
                        shift
                        ;;
                esac
            done
            _kfz__resource_types "$resource_namespace" "${resource_query_parts[*]}"
            ;;
        help|-h|--help)
            _kfz__help
            ;;
        '')
            _kfz__resource_types
            ;;
        *)
            _kfz__resource_picker "$@"
            ;;
    esac
}

# fzf launches execute/preview/transform commands through its configured shell.
# Export only functions that child shells actually need.
export -f \
    _kfz__require_commands \
    _kfz__pause \
    _kfz__preview_render \
    _kfz__render \
    _kfz__page_file \
    _kfz__clip \
    _kfz__resource_meta \
    _kfz__current_namespace \
    _kfz__preview_resource \
    _kfz__select_container \
    _kfz__list \
    _kfz__reload \
    _kfz__toggle_scope \
    _kfz__action \
    _kfz__resource_header \
    _kfz__resource_picker \
    _kfz__resource_types_list \
    _kfz__resource_type_header \
    _kfz__resource_types \
    _kfz__namespace_gate_action \
    _kfz__namespaces_list \
    _kfz__namespace_header \
    _kfz__namespace_action \
    _kfz__context_action \
    _kfz__contexts_list \
    _kfz__context_preview \
    _kfz__context_header \
    _kfz__contexts \
    kfz

