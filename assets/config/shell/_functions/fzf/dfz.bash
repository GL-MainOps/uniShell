# dfz.bash - fzf-driven Docker and Docker Compose operations for Bash
#
# Purpose:
#   Provide a compact K9s-like terminal UI for the Docker Engine and
#   Docker Compose using only Bash, Docker CLI, Docker Compose and fzf.
#
# Public command:
#   dfz                         choose Docker resource type
#   dfz containers              container browser
#   dfz images                 image browser
#   dfz volumes                volume browser
#   dfz networks               network browser
#   dfz compose                current Compose project's services
#   dfz ctx                    Docker context browser
#
# Launch examples:
#   dfz containers nginx
#   dfz containers -c production
#   dfz compose -f compose.yaml
#   dfz compose -f compose.yaml -f compose.prod.yaml
#   dfz compose -p myproject
#
# Design:
#   - fzf uses --exact by default.
#   - Preview is hidden by default. Your existing Ctrl-/ binding can toggle it.
#   - The UI receives one preformatted display field, so there are no literal
#     "\\t" artifacts and columns stay aligned with long names.
#   - Raw identifiers remain in separate hidden fzf fields and are used for
#     all Docker actions. Presentation formatting can therefore never corrupt
#     the object name used by an action.
#   - fzf child commands run through Bash, and all required helper functions
#     are exported so execute/preview/reload bindings work from child shells.
#
# Optional tools:
#   bat       preferred renderer for JSON/YAML/text
#   wl-copy   Wayland clipboard
#   xclip     X11 clipboard
#   xsel      X11 clipboard
#   pbcopy    macOS clipboard

# -----------------------------------------------------------------------------
# Common helpers
# -----------------------------------------------------------------------------

_dfz__require_commands() {
    local missing=()
    local cmd

    for cmd in docker fzf; do
        command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
    done

    if ((${#missing[@]})); then
        printf 'dfz: missing command(s): %s\n' "${missing[*]}" >&2
        return 127
    fi
}

_dfz__require_compose() {
    _dfz__require_commands || return

    _dfz__docker compose version >/dev/null 2>&1 || {
        printf 'dfz: Docker Compose is unavailable. Try `docker compose version`.\n' >&2
        return 127
    }
}

# Run Docker while optionally forcing a context for this dfz invocation.
# DFZ_CONTEXT never changes the user's persistent Docker configuration.
_dfz__docker() {
    local -a args=()

    if [[ -n ${DFZ_CONTEXT:-} ]]; then
        args+=(--context "$DFZ_CONTEXT")
    fi

    command docker "${args[@]}" "$@"
}

# Build one Docker Compose command from environment-backed invocation state.
# Newlines are used internally because they survive export to fzf child Bash
# processes without requiring eval or shell-quoting tricks.
_dfz__compose() {
    local -a args=()
    local file profile

    if [[ -n ${DFZ_COMPOSE_FILES:-} ]]; then
        while IFS= read -r file; do
            [[ -n $file ]] || continue
            args+=(-f "$file")
        done <<< "$DFZ_COMPOSE_FILES"
    fi

    if [[ -n ${DFZ_PROJECT_NAME:-} ]]; then
        args+=(-p "$DFZ_PROJECT_NAME")
    fi

    if [[ -n ${DFZ_PROJECT_DIRECTORY:-} ]]; then
        args+=(--project-directory "$DFZ_PROJECT_DIRECTORY")
    fi

    if [[ -n ${DFZ_PROFILES:-} ]]; then
        while IFS= read -r profile; do
            [[ -n $profile ]] || continue
            args+=(--profile "$profile")
        done <<< "$DFZ_PROFILES"
    fi

    _dfz__docker compose "${args[@]}" "$@"
}

_dfz__pause() {
    printf '\nPress Enter to return to fzf...' >&2
    IFS= read -r _ </dev/tty
}

# Render preview/action output. bat is used when available; cat is deliberately
# used as the fallback because less/pagers are awkward inside fzf previews.
_dfz__render() {
    local language=${1:-text}

    if command -v bat >/dev/null 2>&1; then
        if [[ $language == text ]]; then
            bat --style=plain --color=always --paging=never 2>/dev/null
        else
            bat --style=plain --color=always --paging=never \
                --language="$language" 2>/dev/null
        fi
    else
        cat
    fi
}

# Page action output after Enter/Ctrl-Y. A pipe into bat is intentionally not
# used here: bat treats piped input as non-interactive. Writing the result to a
# temporary file allows --paging=always to invoke the configured pager.
_dfz__page_file() {
    local language=${1:-text}
    local file=$2

    [[ -r $file ]] || return 1

    if command -v bat >/dev/null 2>&1; then
        if [[ $language == text ]]; then
            bat --style=plain --color=always --paging=always "$file"
        else
            bat --style=plain --color=always --paging=always \
                --language="$language" "$file"
        fi
    elif command -v less >/dev/null 2>&1; then
        less -R "$file"
    else
        cat "$file"
    fi
}

# Interactive renderer for actions launched by Enter/Ctrl-Y.  bat only uses
# an interactive pager when its input is a terminal/file; stdin from a pipe is
# treated as non-interactive.  Materialize the action output into a temporary
# file first, then let bat page that file.  less is the secondary fallback and
# cat is the final fallback.
_dfz__page() {
    local language=${1:-text}
    local tmp rc=0

    tmp=$(mktemp "${TMPDIR:-/tmp}/dfz.XXXXXX") || {
        cat
        return 0
    }

    cat > "$tmp"

    if command -v bat >/dev/null 2>&1; then
        if [[ $language == text ]]; then
            bat --style=plain --color=always --paging=always "$tmp" || rc=$?
        else
            bat --style=plain --color=always --paging=always \
                --language="$language" "$tmp" || rc=$?
        fi
    elif command -v less >/dev/null 2>&1; then
        less -R "$tmp" || rc=$?
    else
        cat "$tmp" || rc=$?
    fi

    rm -f "$tmp"
    return "$rc"
}

_dfz__clip() {
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

_dfz__confirm() {
    local prompt=$1
    local answer

    read -r -p "$prompt [y/N] " answer </dev/tty
    [[ $answer =~ ^[Yy]$ ]]
}

# -----------------------------------------------------------------------------
# Docker resource listings
# -----------------------------------------------------------------------------

# Internal container row:
#   ID<TAB>NAME<TAB>IMAGE<TAB>STATE<TAB>STATUS<TAB>DISPLAY
_dfz__containers_list() {
    _dfz__docker ps -a --format '{{.ID}}{{"\t"}}{{.Names}}{{"\t"}}{{.Image}}{{"\t"}}{{.State}}{{"\t"}}{{.Status}}{{"\t"}}{{.Ports}}' 2>/dev/null |
        awk -F '\t' '
            {
                id[++n] = $1
                name[n] = $2
                image[n] = $3
                state[n] = $4
                status[n] = $5
                ports[n] = $6

                if (length(name[n]) > w1) w1 = length(name[n])
                if (length(image[n]) > w2) w2 = length(image[n])
                if (length(state[n]) > w3) w3 = length(state[n])
                if (length(status[n]) > w4) w4 = length(status[n])
            }
            END {
                for (i = 1; i <= n; i++)
                    printf "%s\t%s\t%s\t%s\t%s\t%-*s  %-*s  %-*s  %-*s  %s\n", \
                        id[i], name[i], image[i], state[i], status[i], \
                        w1, name[i], w2, image[i], w3, state[i], w4, status[i], ports[i]
            }
        '
}

# Image row:
#   ID<TAB>REPOSITORY<TAB>TAG<TAB>SIZE<TAB>CREATED<TAB>DISPLAY
_dfz__images_list() {
    _dfz__docker image ls --no-trunc --format '{{.ID}}{{"\t"}}{{.Repository}}{{"\t"}}{{.Tag}}{{"\t"}}{{.Size}}{{"\t"}}{{.CreatedSince}}' 2>/dev/null |
        awk -F '\t' '
            {
                id[++n] = $1
                repo[n] = $2
                tag[n] = $3
                size[n] = $4
                created[n] = $5

                if (length(repo[n]) > w1) w1 = length(repo[n])
                if (length(tag[n]) > w2) w2 = length(tag[n])
                if (length(size[n]) > w3) w3 = length(size[n])
                if (length(created[n]) > w4) w4 = length(created[n])
            }
            END {
                for (i = 1; i <= n; i++)
                    printf "%s\t%s\t%s\t%s\t%s\t%-*s  %-*s  %-*s  %-*s\n", \
                        id[i], repo[i], tag[i], size[i], created[i], \
                        w1, repo[i], w2, tag[i], w3, size[i], w4, created[i]
            }
        '
}

# Volume row:
#   NAME<TAB>DRIVER<TAB>SCOPE<TAB>DISPLAY
_dfz__volumes_list() {
    _dfz__docker volume ls --format '{{.Name}}{{"\t"}}{{.Driver}}{{"\t"}}{{.Scope}}' 2>/dev/null |
        awk -F '\t' '
            {
                name[++n] = $1
                driver[n] = $2
                scope[n] = $3

                if (length(name[n]) > w1) w1 = length(name[n])
                if (length(driver[n]) > w2) w2 = length(driver[n])
                if (length(scope[n]) > w3) w3 = length(scope[n])
            }
            END {
                for (i = 1; i <= n; i++)
                    printf "%s\t%s\t%s\t%-*s  %-*s  %-*s\n", \
                        name[i], driver[i], scope[i], w1, name[i], \
                        w2, driver[i], w3, scope[i]
            }
        '
}

# Network row:
#   ID<TAB>NAME<TAB>DRIVER<TAB>SCOPE<TAB>DISPLAY
_dfz__networks_list() {
    _dfz__docker network ls --format '{{.ID}}{{"\t"}}{{.Name}}{{"\t"}}{{.Driver}}{{"\t"}}{{.Scope}}' 2>/dev/null |
        awk -F '\t' '
            {
                id[++n] = $1
                name[n] = $2
                driver[n] = $3
                scope[n] = $4

                if (length(name[n]) > w1) w1 = length(name[n])
                if (length(driver[n]) > w2) w2 = length(driver[n])
                if (length(scope[n]) > w3) w3 = length(scope[n])
            }
            END {
                for (i = 1; i <= n; i++)
                    printf "%s\t%s\t%s\t%s\t%-*s  %-*s  %-*s\n", \
                        id[i], name[i], driver[i], scope[i], w1, name[i], \
                        w2, driver[i], w3, scope[i]
            }
        '
}

# Compose service/container row:
#   SERVICE<TAB>CONTAINER<TAB>STATE<TAB>HEALTH<TAB>IMAGE<TAB>PORTS<TAB>DISPLAY
#
# `docker compose ps --all` intentionally drives this list. It shows both
# running and stopped containers and is the Compose-supported snapshot of the
# current project's service state.
_dfz__compose_list() {
    _dfz__compose ps --all --format '{{.Service}}{{"\t"}}{{.Name}}{{"\t"}}{{.State}}{{"\t"}}{{.Health}}{{"\t"}}{{.Image}}{{"\t"}}{{.Ports}}' 2>/dev/null |
        awk -F '\t' '
            {
                service[++n] = $1
                container[n] = $2
                state[n] = $3
                health[n] = ($4 == "" ? "-" : $4)
                image[n] = $5
                ports[n] = $6

                if (length(service[n]) > w1) w1 = length(service[n])
                if (length(container[n]) > w2) w2 = length(container[n])
                if (length(state[n]) > w3) w3 = length(state[n])
                if (length(health[n]) > w4) w4 = length(health[n])
                if (length(image[n]) > w5) w5 = length(image[n])
            }
            END {
                for (i = 1; i <= n; i++)
                    printf "%s\t%s\t%s\t%s\t%s\t%s\t%-*s  %-*s  %-*s  %-*s  %-*s  %s\n", \
                        service[i], container[i], state[i], health[i], image[i], ports[i], \
                        w1, service[i], w2, container[i], w3, state[i], w4, health[i], \
                        w5, image[i], ports[i]
            }
        '
}

# -----------------------------------------------------------------------------
# Docker context listing
# -----------------------------------------------------------------------------

_dfz__contexts_list() {
    local current
    current=$(_dfz__docker context show 2>/dev/null || true)

    _dfz__docker context ls --format '{{.Name}}{{"\t"}}{{.Description}}{{"\t"}}{{.DockerEndpoint}}{{"\t"}}{{.Orchestrator}}' 2>/dev/null |
        awk -F '\t' -v current="$current" '
            {
                name[++n] = $1
                desc[n] = ($2 == "" ? "-" : $2)
                endpoint[n] = ($3 == "" ? "-" : $3)
                orchestrator[n] = ($4 == "" ? "-" : $4)
                mark[n] = ($1 == current ? "*" : " ")

                if (length(name[n]) > w1) w1 = length(name[n])
                if (length(desc[n]) > w2) w2 = length(desc[n])
                if (length(endpoint[n]) > w3) w3 = length(endpoint[n])
            }
            END {
                for (i = 1; i <= n; i++)
                   printf "%s\t%s\t%s\t%s\t%s\t%s %-*s  %-*s  %-*s  %s\n", \
                       name[i], desc[i], endpoint[i], orchestrator[i], mark[i], mark[i], \
                       w1, name[i], w2, desc[i], w3, endpoint[i], orchestrator[i]
            }
        '
}

# -----------------------------------------------------------------------------
# Preview functions
# -----------------------------------------------------------------------------

_dfz__preview_container() {
    local name=$1
    [[ -n $name ]] || return 0
    _dfz__docker inspect "$name" 2>&1 | _dfz__render json
}

_dfz__preview_image() {
    local id=$1
    [[ -n $id ]] || return 0
    _dfz__docker image inspect "$id" 2>&1 | _dfz__render json
}

_dfz__preview_volume() {
    local name=$1
    [[ -n $name ]] || return 0
    _dfz__docker volume inspect "$name" 2>&1 | _dfz__render json
}

_dfz__preview_network() {
    local name=$1
    [[ -n $name ]] || return 0
    _dfz__docker network inspect "$name" 2>&1 | _dfz__render json
}

_dfz__preview_compose_service() {
    local service=$1
    [[ -n $service ]] || return 0

    # Prefer the service's actual containers. If the service has not yet been
    # created, fall back to the resolved Compose model so preview is not empty.
    local result
    result=$(_dfz__compose ps --all --format json "$service" 2>&1)
    if [[ -n $result && $result != '[]' ]]; then
        printf '%s\n' "$result" | _dfz__render json
    else
        _dfz__compose config --format json 2>&1 | _dfz__render json
    fi
}

_dfz__preview_context() {
    local context=$1
    [[ -n $context ]] || return 0
    _dfz__docker context inspect "$context" 2>&1 | _dfz__render json
}

# -----------------------------------------------------------------------------
# Action dispatcher
# -----------------------------------------------------------------------------

_dfz__action() {
    local action=$1
    local resource=$2

    [[ ${3-} == --file ]] || {
        printf 'dfz: internal error: expected --file <fzf temp file>\n' >&2
        return 2
    }

    local file=${4-}
    [[ -r $file ]] || {
        printf 'dfz: selection file is not readable\n' >&2
        return 2
    }

    local -a f1=() f2=() f3=() f4=() f5=() f6=()
    local a b c d e f

    while IFS=$'\t' read -r a b c d e f; do
        [[ -n $a ]] || continue
        f1+=("$a")
        f2+=("$b")
        f3+=("$c")
        f4+=("$d")
        f5+=("$e")
        f6+=("$f")
    done < "$file"

    ((${#f1[@]})) || return 0

    local i

    case $resource in
        containers)
            case $action in
                inspect|yaml)
                    local page_file
                    page_file=$(mktemp "${TMPDIR:-/tmp}/dfz-inspect.XXXXXX") || return 1
                    for i in "${!f2[@]}"; do
                        printf '\n===== %s =====\n' "${f2[i]}" >> "$page_file"
                        _dfz__docker inspect "${f2[i]}" >> "$page_file" 2>&1
                    done
                    _dfz__page_file json "$page_file"
                    local page_rc=$?
                    rm -f "$page_file"
                    return "$page_rc"
                    ;;
                delete)
                    printf 'Selected container(s):\n'
                    for i in "${!f2[@]}"; do printf '  %s\n' "${f2[i]}"; done
                    printf '\n'
                    _dfz__confirm "Remove ${#f2[@]} container(s)?" || return 0
                    _dfz__docker rm "${f2[@]}"
                    ;;
                logs)
                    for i in "${!f2[@]}"; do
                        printf '\n===== %s =====\n' "${f2[i]}"
                        _dfz__docker logs --tail 200 "${f2[i]}"
                    done
                    _dfz__pause
                    ;;
                logs-follow)
                    ((${#f2[@]} == 1)) || {
                        printf 'dfz: follow logs requires exactly one selected container.\n' >&2
                        _dfz__pause
                        return 1
                    }
                    _dfz__docker logs -f --tail 200 "${f2[0]}"
                    ;;
                exec)
                    ((${#f2[@]} == 1)) || {
                        printf 'dfz: exec requires exactly one selected container.\n' >&2
                        _dfz__pause
                        return 1
                    }
                    if _dfz__docker exec "${f2[0]}" /bin/bash -c 'exit 0' >/dev/null 2>&1; then
                        _dfz__docker exec -it "${f2[0]}" /bin/bash -il
                    else
                        _dfz__docker exec -it "${f2[0]}" /bin/sh
                    fi
                    ;;
                attach)
                    ((${#f2[@]} == 1)) || {
                        printf 'dfz: attach requires exactly one selected container.\n' >&2
                        _dfz__pause
                        return 1
                    }
                    _dfz__docker attach "${f2[0]}"
                    ;;
                restart)
                    _dfz__docker restart "${f2[@]}"
                    ;;
                toggle-start-stop)
                    for i in "${!f2[@]}"; do
                        if [[ ${f4[i]} == running ]]; then
                            _dfz__docker stop "${f2[i]}"
                        else
                            _dfz__docker start "${f2[i]}"
                        fi
                    done
                    ;;
                toggle-pause)
                    for i in "${!f2[@]}"; do
                        local paused
                        paused=$(_dfz__docker inspect -f '{{.State.Paused}}' "${f2[i]}" 2>/dev/null)
                        if [[ $paused == true ]]; then
                            _dfz__docker unpause "${f2[i]}"
                        else
                            _dfz__docker pause "${f2[i]}"
                        fi
                    done
                    ;;
                top)
                    for i in "${!f2[@]}"; do
                        printf '\n===== %s =====\n' "${f2[i]}"
                        _dfz__docker top "${f2[i]}"
                    done
                    _dfz__pause
                    ;;
                stats)
                    _dfz__docker stats --no-stream "${f2[@]}"
                    _dfz__pause
                    ;;
                ports)
                    for i in "${!f2[@]}"; do
                        printf '\n===== %s =====\n' "${f2[i]}"
                        _dfz__docker port "${f2[i]}"
                    done
                    _dfz__pause
                    ;;
                copy-name)
                    printf '%s\n' "${f2[@]}" | _dfz__clip
                    ;;
                copy-id)
                    printf '%s\n' "${f1[@]}" | _dfz__clip
                    ;;
                prune)
                   _dfz__confirm 'Prune all stopped containers?' || return 0
                   _dfz__docker container prune -f
                   ;;
            esac
            ;;

        images)
            case $action in
                inspect|yaml)
                    local page_file
                    page_file=$(mktemp "${TMPDIR:-/tmp}/dfz-inspect.XXXXXX") || return 1
                    for i in "${!f1[@]}"; do
                        printf '\n===== %s:%s =====\n' "${f2[i]}" "${f3[i]}" >> "$page_file"
                        _dfz__docker image inspect "${f1[i]}" >> "$page_file" 2>&1
                    done
                    _dfz__page_file json "$page_file"
                    local page_rc=$?
                    rm -f "$page_file"
                    return "$page_rc"
                    ;;
                delete)
                    printf 'Selected image(s):\n'
                    for i in "${!f1[@]}"; do printf '  %s:%s\n' "${f2[i]}" "${f3[i]}"; done
                    printf '\n'
                    _dfz__confirm "Remove ${#f1[@]} image(s)?" || return 0
                    _dfz__docker image rm "${f1[@]}"
                    ;;
                history)
                    for i in "${!f1[@]}"; do
                        printf '\n===== %s:%s =====\n' "${f2[i]}" "${f3[i]}"
                        _dfz__docker image history "${f1[i]}"
                    done
                    _dfz__pause
                    ;;
                copy-name)
                    for i in "${!f2[@]}"; do printf '%s:%s\n' "${f2[i]}" "${f3[i]}"; done | _dfz__clip
                    ;;
                copy-id)
                    printf '%s\n' "${f1[@]}" | _dfz__clip
                    ;;
            esac
            ;;

        volumes)
            case $action in
                inspect|yaml)
                    local page_file
                    page_file=$(mktemp "${TMPDIR:-/tmp}/dfz-inspect.XXXXXX") || return 1
                    for i in "${!f1[@]}"; do
                        printf '\n===== %s =====\n' "${f1[i]}" >> "$page_file"
                        _dfz__docker volume inspect "${f1[i]}" >> "$page_file" 2>&1
                    done
                    _dfz__page_file json "$page_file"
                    local page_rc=$?
                    rm -f "$page_file"
                    return "$page_rc"
                    ;;
                delete)
                    printf 'Selected volume(s):\n'
                    for i in "${!f1[@]}"; do printf '  %s\n' "${f1[i]}"; done
                    printf '\n'
                    _dfz__confirm "Remove ${#f1[@]} volume(s)?" || return 0
                    _dfz__docker volume rm "${f1[@]}"
                    ;;
                copy-name)
                    printf '%s\n' "${f1[@]}" | _dfz__clip
                    ;;
            esac
            ;;

        networks)
            case $action in
                inspect|yaml)
                    local page_file
                    page_file=$(mktemp "${TMPDIR:-/tmp}/dfz-inspect.XXXXXX") || return 1
                    for i in "${!f2[@]}"; do
                        printf '\n===== %s =====\n' "${f2[i]}" >> "$page_file"
                        _dfz__docker network inspect "${f2[i]}" >> "$page_file" 2>&1
                    done
                    _dfz__page_file json "$page_file"
                    local page_rc=$?
                    rm -f "$page_file"
                    return "$page_rc"
                    ;;
                delete)
                    printf 'Selected network(s):\n'
                    for i in "${!f2[@]}"; do printf '  %s\n' "${f2[i]}"; done
                    printf '\n'
                    _dfz__confirm "Remove ${#f2[@]} network(s)?" || return 0
                    _dfz__docker network rm "${f2[@]}"
                    ;;
                copy-name)
                    printf '%s\n' "${f2[@]}" | _dfz__clip
                    ;;
            esac
            ;;

        compose)
            case $action in
                inspect)
                    local page_file
                    page_file=$(mktemp "${TMPDIR:-/tmp}/dfz-inspect.XXXXXX") || return 1
                    for i in "${!f1[@]}"; do
                        printf '\n===== service: %s =====\n' "${f1[i]}" >> "$page_file"
                        _dfz__compose ps --all --format json "${f1[i]}" >> "$page_file" 2>&1
                    done
                    _dfz__page_file json "$page_file"
                    local page_rc=$?
                    rm -f "$page_file"
                    return "$page_rc"
                    ;;
                config)
                    _dfz__compose config --format yaml | _dfz__page_file yaml /dev/stdin
                    _dfz__pause
                    ;;
                delete)
                    printf 'Selected Compose service(s):\n'
                    for i in "${!f1[@]}"; do printf '  %s\n' "${f1[i]}"; done
                    printf '\nOnly stopped service containers are removed by `docker compose rm`.\n'
                    _dfz__confirm "Remove selected stopped service container(s)?" || return 0
                    _dfz__compose rm -f "${f1[@]}"
                    ;;
                logs)
                    for i in "${!f1[@]}"; do
                        printf '\n===== %s =====\n' "${f1[i]}"
                        _dfz__compose logs --tail 200 "${f1[i]}"
                    done
                    _dfz__pause
                    ;;
                logs-follow)
                    ((${#f1[@]} == 1)) || {
                        printf 'dfz: follow logs requires exactly one selected service.\n' >&2
                        _dfz__pause
                        return 1
                    }
                    _dfz__compose logs -f --tail 200 "${f1[0]}"
                    ;;
                exec)
                    ((${#f1[@]} == 1)) || {
                        printf 'dfz: exec requires exactly one selected service.\n' >&2
                        _dfz__pause
                        return 1
                    }
                   local idx; read -r -p 'Replica index [1]: ' idx </dev/tty; idx=${idx:-1}
                   if _dfz__compose exec --index "$idx" "${f1[0]}" /bin/bash -c 'exit 0' >/dev/null 2>&1; then
                        _dfz__compose exec --index "$idx" "${f1[0]}" /bin/bash -il
                    else
                        _dfz__compose exec --index "$idx" "${f1[0]}" /bin/sh
                    fi
                    ;;
                restart)
                    _dfz__compose restart "${f1[@]}"
                    ;;
                toggle-start-stop)
                    for i in "${!f1[@]}"; do
                        case ${f3[i]} in
                            running|paused|restarting)
                                _dfz__compose stop "${f1[i]}"
                                ;;
                            *)
                                _dfz__compose start "${f1[i]}"
                                ;;
                        esac
                    done
                    ;;
                pause)
                    _dfz__compose pause "${f1[@]}"
                    ;;
                unpause)
                    _dfz__compose unpause "${f1[@]}"
                    ;;
                top)
                    for i in "${!f1[@]}"; do
                        printf '\n===== %s =====\n' "${f1[i]}"
                        _dfz__compose top "${f1[i]}"
                    done
                    _dfz__pause
                    ;;
                ports)
                    ((${#f1[@]} == 1)) || {
                        printf 'dfz: port lookup requires exactly one selected service.\n' >&2
                        _dfz__pause
                        return 1
                    }
                    local port
                    read -r -p 'Container port (example: 80): ' port </dev/tty
                    [[ -n $port ]] || return 0
                    _dfz__compose port "${f1[0]}" "$port"
                    _dfz__pause
                    ;;
                copy-name)
                    printf '%s\n' "${f1[@]}" | _dfz__clip
                    ;;
                copy-container)
                    printf '%s\n' "${f2[@]}" | _dfz__clip
                    ;;
                scale)
                    ((${#f1[@]} == 1)) || {
                        printf 'dfz: scale requires exactly one selected service.\n' >&2
                        _dfz__pause
                        return 1
                    }
                    local replicas
                    read -r -p 'Desired replicas: ' replicas </dev/tty
                    [[ $replicas =~ ^[0-9]+$ ]] || {
                        printf 'dfz: replicas must be a non-negative integer.\n' >&2
                        _dfz__pause
                        return 1
                    }
                    _dfz__compose up -d --scale "${f1[0]}=$replicas" "${f1[0]}"
                    ;;
                pull)
                    _dfz__compose pull "${f1[@]}"
                    ;;
                build)
                    _dfz__compose build "${f1[@]}"
                    ;;
            esac
            ;;
    esac
}

# -----------------------------------------------------------------------------
# Headers
# -----------------------------------------------------------------------------

_dfz__header() {
    local resource=$1

    printf '%s\n%s' \
        'Enter inspect · Tab select · Ctrl-D delete · Ctrl-R reload · Ctrl-Y JSON' \
        'Alt-C copy · Ctrl-/ preview · exact matching'

    case $resource in
        containers)
            printf '\n%s\n%s' \
                'Container: Ctrl-L logs · Alt-L follow · Ctrl-X exec · Alt-A attach' \
                'Container: Alt-R restart · Alt-S start/stop · Alt-P pause · Ctrl-T top · Ctrl-O stats · Alt-W ports'
            ;;
        images)
            printf '\n%s' 'Image: Alt-U history · Ctrl-D remove'
            ;;
        volumes)
            printf '\n%s' 'Volume: Ctrl-D remove'
            ;;
        networks)
            printf '\n%s' 'Network: Ctrl-D remove'
            ;;
        compose)
            printf '\n%s\n%s' \
                'Compose: Ctrl-L logs · Alt-L follow · Ctrl-X exec · Alt-R restart' \
                'Compose: Alt-S start/stop · Alt-P pause · Alt-U unpause · Ctrl-T top · Alt-W ports · Alt-H scale'
            ;;
    esac
}

# -----------------------------------------------------------------------------
# Generic resource picker
# -----------------------------------------------------------------------------

_dfz__resource_picker() {
    local resource=$1
    shift || true

    local initial_query=''
    (($#)) && initial_query=$*

    local list_cmd preview_cmd
    local -a args=(
        --with-shell='bash -c'
        --exact
        --multi
        --delimiter=$'\t'
        --prompt="Docker:$resource ❯❯❯ "
        --query="$initial_query"
        --header="$(_dfz__header "$resource")"
        --preview-window='right:55%:hidden:wrap'
        --bind="ctrl-r:reload(_dfz__${resource}_list)"
        --bind="enter:execute(_dfz__action inspect '$resource' --file {+f})"
        --bind="ctrl-d:execute(_dfz__action delete '$resource' --file {+f})+reload(_dfz__${resource}_list)"
        --bind="ctrl-y:execute(_dfz__action yaml '$resource' --file {+f})"
        --bind="alt-c:execute-silent(_dfz__action copy-name '$resource' --file {+f})"
    )

    case $resource in
        containers)
            list_cmd='_dfz__containers_list'
            preview_cmd='_dfz__preview_container {2}'
            args+=(
                --with-nth=6
                --accept-nth='{2}'
                --preview="$preview_cmd"
                --bind='ctrl-l:execute(_dfz__action logs containers --file {+f})'
                --bind='alt-l:execute(_dfz__action logs-follow containers --file {+f})'
                --bind='ctrl-x:execute(_dfz__action exec containers --file {+f})'
                --bind='alt-a:execute(_dfz__action attach containers --file {+f})'
                --bind='alt-r:execute(_dfz__action restart containers --file {+f})+reload(_dfz__containers_list)'
                --bind='alt-s:execute(_dfz__action toggle-start-stop containers --file {+f})+reload(_dfz__containers_list)'
                --bind='alt-p:execute(_dfz__action toggle-pause containers --file {+f})+reload(_dfz__containers_list)'
                --bind='ctrl-t:execute(_dfz__action top containers --file {+f})'
                --bind='ctrl-o:execute(_dfz__action stats containers --file {+f})'
                --bind='alt-w:execute(_dfz__action ports containers --file {+f})'
                --bind='alt-i:execute-silent(_dfz__action copy-id containers --file {+f})'
                --bind='ctrl-p:execute(_dfz__action prune containers --file {+f})+reload(_dfz__containers_list)'
            )
            ;;
        images)
            list_cmd='_dfz__images_list'
            preview_cmd='_dfz__preview_image {1}'
            args+=(
                --with-nth=6
                --accept-nth='{1}'
                --preview="$preview_cmd"
                --bind='alt-u:execute(_dfz__action history images --file {+f})'
                --bind='alt-c:execute-silent(_dfz__action copy-name images --file {+f})'
                --bind='alt-i:execute-silent(_dfz__action copy-id images --file {+f})'
            )
            ;;
        volumes)
            list_cmd='_dfz__volumes_list'
            preview_cmd='_dfz__preview_volume {1}'
            args+=(
                --with-nth=4
                --accept-nth='{1}'
                --preview="$preview_cmd"
            )
            ;;
        networks)
            list_cmd='_dfz__networks_list'
            preview_cmd='_dfz__preview_network {2}'
            args+=(
                --with-nth=5
                --accept-nth='{2}'
                --preview="$preview_cmd"
            )
            ;;
        compose)
            list_cmd='_dfz__compose_list'
            preview_cmd='_dfz__preview_compose_service {1}'
            args+=(
                --with-nth=7
                --accept-nth='{1}'
                --prompt='Compose service ❯❯❯ '
                --preview="$preview_cmd"
                --bind='ctrl-l:execute(_dfz__action logs compose --file {+f})'
                --bind='alt-l:execute(_dfz__action logs-follow compose --file {+f})'
                --bind='ctrl-x:execute(_dfz__action exec compose --file {+f})'
                --bind='alt-r:execute(_dfz__action restart compose --file {+f})+reload(_dfz__compose_list)'
                --bind='alt-s:execute(_dfz__action toggle-start-stop compose --file {+f})+reload(_dfz__compose_list)'
                --bind='alt-p:execute(_dfz__action pause compose --file {+f})+reload(_dfz__compose_list)'
                --bind='alt-u:execute(_dfz__action unpause compose --file {+f})+reload(_dfz__compose_list)'
                --bind='ctrl-t:execute(_dfz__action top compose --file {+f})'
                --bind='alt-w:execute(_dfz__action ports compose --file {+f})'
                --bind='alt-h:execute(_dfz__action scale compose --file {+f})+reload(_dfz__compose_list)'
                --bind='alt-g:execute(_dfz__action pull compose --file {+f})' \
                --bind='alt-b:execute(_dfz__action build compose --file {+f})+reload(_dfz__compose_list)' \
                --bind='alt-c:execute-silent(_dfz__action copy-name compose --file {+f})'
                --bind='alt-i:execute-silent(_dfz__action copy-container compose --file {+f})'
            )
            ;;
        *)
            printf 'dfz: unsupported resource type: %s\n' "$resource" >&2
            return 2
            ;;
    esac

    "$list_cmd" | fzf "${args[@]}"
}

# -----------------------------------------------------------------------------
# Resource type chooser
# -----------------------------------------------------------------------------

_dfz__resource_types_list() {
    cat <<'EOF'
containers	container	Docker Engine	Container
images	image	Docker Engine	Image
volumes	volume	Docker Engine	Volume
networks	network	Docker Engine	Network
compose	compose	Docker Compose (current project)	Service
ctx	context	Docker CLI	Context
EOF
}

_dfz__resource_types() {
    local selected resource initial_query=${1:-}

    selected=$(
        _dfz__resource_types_list |
            fzf \
                --with-shell='bash -c' \
                --exact \
                --no-multi \
                --delimiter=$'\t' \
                --with-nth=4 \
                --accept-nth='{1}' \
                --prompt='Docker resource ❯❯❯ ' \
                --query="$initial_query" \
                --header='Enter browse · Ctrl-R reload · Ctrl-/ preview · exact matching' \
                --preview-window='right:55%:hidden:wrap' \
                --preview='printf "Type: %s\\nBackend: %s\\nKind: %s\\n" {1} {3} {4}' \
                --bind='ctrl-r:reload(_dfz__resource_types_list)' \
                --bind='alt-c:execute-silent(printf %s {1} | _dfz__clip)'
    ) || return 0

    resource=$selected
    [[ -n $resource ]] || return 0

    if [[ $resource == compose ]]; then
        _dfz__require_compose || return
    elif [[ $resource == ctx ]]; then
        _dfz__contexts
        return $?
    fi

    _dfz__resource_picker "$resource"
}

# -----------------------------------------------------------------------------
# Docker context picker
# -----------------------------------------------------------------------------

_dfz__context_header() {
    printf '%s\n%s' \
        'Enter use context · Ctrl-R reload · Ctrl-D delete · Alt-C copy' \
        'Docker context changes the persistent Docker CLI default · Ctrl-/ preview'
}

_dfz__contexts() {
    local selected

    selected=$(
        _dfz__contexts_list |
            fzf \
                --with-shell='bash -c' \
                --exact \
                --no-multi \
                --delimiter=$'\t' \
                --with-nth=6 \
                --accept-nth='{1}' \
                --prompt='Docker context ❯❯❯ ' \
                --header="$(_dfz__context_header)" \
                --preview='_dfz__preview_context {1}' \
                --preview-window='right:55%:hidden:wrap' \
                --bind='ctrl-r:reload(_dfz__contexts_list)' \
                --bind='ctrl-d:execute(_dfz__context_delete {1})+reload(_dfz__contexts_list)' \
                --bind='alt-c:execute-silent(printf %s {1} | _dfz__clip)'
    ) || return 0

    [[ -n $selected ]] || return 0

    # fzf ran in a child process, so perform the actual parent-shell operation
    # here. This intentionally changes Docker's persistent default context,
    # matching `docker context use` semantics.
    _dfz__docker context use "$selected"
}

_dfz__context_delete() {
    local context=$1
    [[ -n $context ]] || return 0

    printf 'Context: %s\n' "$context"
    _dfz__confirm 'Delete this Docker context?' || return 0
    _dfz__docker context rm "$context"
}

# -----------------------------------------------------------------------------
# Argument parsing
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

_dfz__help() {
    cat <<'EOF'
dfz - fzf-driven Docker / Docker Compose operations

Usage:
  dfz                         Choose a Docker resource type.
  dfz containers [QUERY]      Browse all containers, running and stopped.
  dfz images [QUERY]          Browse local images.
  dfz volumes [QUERY]         Browse Docker volumes.
  dfz networks [QUERY]        Browse Docker networks.
  dfz compose [QUERY]         Browse services in the current Compose project.
  dfz ctx                     Browse and switch Docker contexts.

Docker context:
  dfz -c NAME containers      Use NAME for this dfz invocation.
  dfz --context NAME ...      Same as above.

Compose options:
  dfz compose -f compose.yml
  dfz compose -f base.yml -f prod.yml
  dfz compose -p PROJECT
  dfz compose --project-directory DIR
  dfz compose --profile PROFILE

Global picker keys:
  Enter       inspect/describe selected item(s)
  Tab         select multiple
  Ctrl-D      delete/remove (confirmation where destructive)
  Ctrl-R      reload
  Ctrl-Y      JSON / Compose config
  Alt-C       copy primary name
  Ctrl-/      toggle preview; preview starts hidden

Containers:
  Ctrl-L      logs
  Alt-L       follow logs (one container)
  Ctrl-X      exec shell (one container)
  Alt-A       attach (one container)
  Alt-R       restart
  Alt-S       start/stop toggle
  Alt-P       pause/unpause toggle
  Ctrl-T      top
  Ctrl-O      stats
  Alt-W       published ports
  Alt-I       copy ID

Images:
  Alt-U       history
  Ctrl-D      remove
  Alt-C       copy repository:tag
  Alt-I       copy image ID

Compose services:
  Ctrl-L      logs
  Alt-L       follow logs (one service)
  Ctrl-X      exec shell (one service)
  Alt-R       restart
  Alt-S       start/stop
  Alt-P       pause
  Alt-U       unpause
  Ctrl-T      top
  Alt-W       port lookup
  Alt-H       scale (one service)

Matching:
  All fzf pickers use exact matching by default.
EOF
}

# -----------------------------------------------------------------------------
# Public entry point
# -----------------------------------------------------------------------------

dfz() {
    # These are invocation-local state variables. They are deliberately reset
    # for every call so `dfz -c production ...` cannot leak into later calls.
    DFZ_CONTEXT=''
    DFZ_COMPOSE_FILES=''
    DFZ_PROJECT_NAME=''
    DFZ_PROJECT_DIRECTORY=''
    DFZ_PROFILES=''
    export DFZ_CONTEXT DFZ_COMPOSE_FILES DFZ_PROJECT_NAME DFZ_PROJECT_DIRECTORY DFZ_PROFILES

    local -a args=()
    local -a query=()
    local arg resource

    # Parse global Docker context options before the resource type.
    while (($#)); do
        arg=$1
        case $arg in
            -c|--context)
                (($# >= 2)) || {
                    printf 'dfz: %s requires a context name.\n' "$arg" >&2
                    return 2
                }
                DFZ_CONTEXT=$2
                shift 2
                ;;
            --)
                shift
                args+=("$@")
                break
                ;;
            *)
                args+=("$arg")
                shift
                ;;
        esac
    done

    resource=${args[0]-}
    args=("${args[@]:1}")

    case $resource in
        help|-h|--help)
            _dfz__help
            return 0
            ;;
    esac

    _dfz__require_commands || return

    case $resource in
        '')
            _dfz__resource_types
            ;;

        ctx|context|contexts)
            _dfz__contexts
            ;;

        containers|container|ps|images|image|img|volumes|volume|vol|networks|network|net)
            # The Docker engine resource commands accept a query and an optional
            # context. Keep the option parser independent of resource type.
            while ((${#args[@]})); do
                arg=${args[0]}
                args=("${args[@]:1}")
                case $arg in
                    -c|--context)
                        ((${#args[@]})) || {
                            printf 'dfz: %s requires a context name.\n' "$arg" >&2
                            return 2
                        }
                        DFZ_CONTEXT=${args[0]}
                        args=("${args[@]:1}")
                        ;;
                    --)
                        query+=("${args[@]}")
                        args=()
                        ;;
                    *)
                        query+=("$arg")
                        ;;
                esac
            done

            case $resource in
                containers|container|ps)
                    _dfz__resource_picker containers "${query[@]}"
                    ;;
                images|image|img)
                    _dfz__resource_picker images "${query[@]}"
                    ;;
                volumes|volume|vol)
                    _dfz__resource_picker volumes "${query[@]}"
                    ;;
                networks|network|net)
                    _dfz__resource_picker networks "${query[@]}"
                    ;;
            esac
            ;;

        compose|composes|services|service)
            # Docker Compose accepts its normal -f/-p/project-directory/profile
            # options here. Global Docker context is already handled above, and
            # -c is also accepted after the `compose` resource.
            while ((${#args[@]})); do
                arg=${args[0]}
                args=("${args[@]:1}")
                case $arg in
                    -c|--context)
                        ((${#args[@]})) || {
                            printf 'dfz: %s requires a context name.\n' "$arg" >&2
                            return 2
                        }
                        DFZ_CONTEXT=${args[0]}
                        args=("${args[@]:1}")
                        ;;
                    -f|--file)
                        ((${#args[@]})) || {
                            printf 'dfz: %s requires a compose file.\n' "$arg" >&2
                            return 2
                        }
                        DFZ_COMPOSE_FILES+="${DFZ_COMPOSE_FILES:+$'\n'}${args[0]}"
                        args=("${args[@]:1}")
                        ;;
                    -p|--project-name)
                        ((${#args[@]})) || {
                            printf 'dfz: %s requires a project name.\n' "$arg" >&2
                            return 2
                        }
                        DFZ_PROJECT_NAME=${args[0]}
                        args=("${args[@]:1}")
                        ;;
                    --project-directory)
                        ((${#args[@]})) || {
                            printf 'dfz: --project-directory requires a directory.\n' >&2
                            return 2
                        }
                        DFZ_PROJECT_DIRECTORY=${args[0]}
                        args=("${args[@]:1}")
                        ;;
                    --profile)
                        ((${#args[@]})) || {
                            printf 'dfz: --profile requires a profile name.\n' >&2
                            return 2
                        }
                        DFZ_PROFILES+="${DFZ_PROFILES:+$'\n'}${args[0]}"
                        args=("${args[@]:1}")
                        ;;
                    --)
                        query+=("${args[@]}")
                        args=()
                        ;;
                    *)
                        query+=("$arg")
                        ;;
                esac
            done

            _dfz__require_compose || return
            _dfz__resource_picker compose "${query[@]}"
            ;;

        resource|resources|type|types)
            query=("${args[@]}")
            _dfz__resource_types "${query[*]}"
            ;;

        *)
            printf 'dfz: unknown resource: %s\n' "$resource" >&2
            printf 'Run `dfz --help` or just `dfz` to choose a resource.\n' >&2
            return 2
            ;;
    esac
}

# Export everything used by fzf child processes. Docker itself and the user's
# existing environment (DOCKER_HOST, DOCKER_CONFIG, DOCKER_CONTEXT, etc.) are
# inherited normally.
export -f \
    _dfz__require_commands \
    _dfz__require_compose \
    _dfz__docker \
    _dfz__compose \
    _dfz__pause \
    _dfz__render \
    _dfz__page_file \
    _dfz__clip \
    _dfz__confirm \
    _dfz__containers_list \
    _dfz__images_list \
    _dfz__volumes_list \
    _dfz__networks_list \
    _dfz__compose_list \
    _dfz__contexts_list \
    _dfz__preview_container \
    _dfz__preview_image \
    _dfz__preview_volume \
    _dfz__preview_network \
    _dfz__preview_compose_service \
    _dfz__preview_context \
    _dfz__action \
    _dfz__header \
    _dfz__resource_picker \
    _dfz__resource_types_list \
    _dfz__resource_types \
    _dfz__context_header \
    _dfz__contexts \
    _dfz__context_delete \
    _dfz__help \
    dfz

