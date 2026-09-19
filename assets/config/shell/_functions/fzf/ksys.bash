# ksys.bash - fzf-driven systemd operations for Bash
#
# Purpose:
#   Provide a compact K9s-like interface for systemd using Bash, systemctl,
#   journalctl, and fzf.
#
# Scope support:
#   system manager (default) and user manager (--user / -u)
#
# Examples:
#   ksys                         # choose system/user + services/timers/logs
#   ksys services               # system services
#   ksys timers                 # system timers
#   ksys logs                   # system journal by unit
#   ksys user services          # user services
#   ksys user timers            # user timers
#   ksys user logs              # user journal by unit
#   ksys --user services        # user services
#   ksys -u timers              # user timers
#
# Design:
#   - fzf matching is EXACT by default.
#   - Preview is hidden by default. Your existing Ctrl-/ toggle-preview binding
#     reveals it.
#   - Every picker keeps private tab-delimited action fields while displaying
#     one pre-formatted field. Internal tabs are never presented literally.
#   - system/user scope is passed explicitly to systemctl/journalctl.
#   - Mutating operations use fzf execute/reload bindings.
#   - journalctl is used directly for journal operations.
#
# Requirements:
#   bash, systemctl, journalctl, fzf
#
# Optional:
#   bat for colored text rendering; cat is the fallback.

# -----------------------------------------------------------------------------
# Common helpers
# -----------------------------------------------------------------------------

_ksys__require_commands() {
    local missing=()
    local cmd

    for cmd in systemctl journalctl fzf; do
        command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
    done

    if ((${#missing[@]})); then
        printf 'ksys: missing command(s): %s\n' "${missing[*]}" >&2
        return 127
    fi
}

_ksys__pause() {
    printf '\nPress Enter to return to fzf...' >&2
    IFS= read -r _ </dev/tty
}

_ksys__render() {
    local language=${1:-text}

    if command -v bat >/dev/null 2>&1; then
        if [[ $language == text ]]; then
            bat --style=plain --color=always --paging=never 2>/dev/null
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

_ksys__preview_render() {
    _ksys__render "${1:-text}"
}

_ksys__clip() {
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

_ksys__scope_args() {
    [[ ${1:-system} == user ]] && printf '%s\n' '--user'
}

_ksys__scope_label() {
    [[ ${1:-system} == user ]] && printf 'user' || printf 'system'
}

# Run a systemctl operation against the requested manager.
#
# System-manager mutations require root on a normal Linux installation. We do
# not blindly prefix every systemctl invocation with sudo: listings, status,
# previews, and reads remain unprivileged. For a mutating operation we first
# refresh sudo credentials, then invoke exactly one sudo command.
#
# User-manager operations deliberately never use sudo: they belong to the
# calling user's systemd --user instance.
#
# Keep the caller's interactive tooling environment when sudo crosses into
# the system manager. In particular, EDITOR is needed by `systemctl edit`,
# while PATH/TMUX/SSH_AUTH_SOCK and the UniShell runtime are useful to editors
# and interactive shell integrations started from the unit editor.
# Keep this literal inside the exported helper. fzf runs execute/preview
# actions in child Bash processes, and an unexported shell variable would not
# exist there. The exact preserve list therefore must not depend on parent-shell
# variable scope.
_ksys__sudo_systemctl() {
    sudo --preserve-env=EDITOR,MANPAGER,PAGER,PATH,TMUX,SSH_AUTH_SOCK,UNISHELL_SESSION_RUNTIME_DIR \
        systemctl "$@"
}

_ksys__systemctl_mutate() {
    local scope=${1:-system}
    shift

    if [[ $scope == user ]]; then
        systemctl --user "$@"
        return
    fi

    command -v sudo >/dev/null 2>&1 || {
        printf 'ksys: sudo is required for system-manager mutations.\n' >&2
        return 127
    }

    sudo -v || {
        printf 'ksys: sudo authentication failed; action not executed.\n' >&2
        return 1
    }

    sudo --preserve-env=EDITOR,MANPAGER,PAGER,PATH,TMUX,SSH_AUTH_SOCK,UNISHELL_SESSION_RUNTIME_DIR \
        systemctl "$@"
}

# -----------------------------------------------------------------------------
# Service picker
# -----------------------------------------------------------------------------

_ksys__services_list() {
    local scope=${1:-system}
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    systemctl "${scope_args[@]}" \
        list-units \
        --type=service \
        --all \
        --no-legend \
        --no-pager \
        --full \
        --plain 2>/dev/null |
        awk '
            {
                unit = $1
                load = $2
                active = $3
                substate = $4
                if (unit == "") next

                description = ""
                if (NF >= 5) {
                    match($0, /^[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]*/)
                    description = substr($0, RSTART + RLENGTH)
                }

                u[++count] = unit
                l[count] = load
                a[count] = active
                s[count] = substate
                d[count] = description

                if (length(unit) > wu) wu = length(unit)
                if (length(load) > wl) wl = length(load)
                if (length(active) > wa) wa = length(active)
                if (length(substate) > ws) ws = length(substate)
            }
            END {
                for (i = 1; i <= count; i++)
                    printf "%s\t%s\t%s\t%s\t%-*s  %-*s  %-*s  %-*s  %s\n", \
                        u[i], l[i], a[i], s[i], \
                        wu, u[i], wl, l[i], wa, a[i], ws, s[i], d[i]
            }
        '
}

_ksys__services_header() {
    printf '%s\n%s\n%s' \
        'Enter status · Tab select · Ctrl-R reload · Ctrl-/ preview' \
        'Ctrl-S start · Ctrl-X stop · Alt-R restart · Alt-L reload · Ctrl-E edit · Ctrl-Y cat' \
        'Ctrl-L logs · Alt-F follow · Alt-P previous boot · Ctrl-N enable · Alt-D disable · Alt-M mask · Alt-U unmask'
}

_ksys__service_preview() {
    local scope=$1 unit=$2
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    systemctl "${scope_args[@]}" status --no-pager --full "$unit" 2>&1 |
        _ksys__preview_render text
}

# -----------------------------------------------------------------------------
# Timer picker
# -----------------------------------------------------------------------------

_ksys__timers_list() {
    local scope=${1:-system}
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    systemctl "${scope_args[@]}" \
        list-timers \
        --all \
        --no-legend \
        --no-pager \
        --full \
        --plain 2>/dev/null |
        awk '
            {
                if (NF < 2) next

                timer = $(NF - 1)
                activates = $NF

                prefix = $0
                sub(/[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]*$/, "", prefix)
                sub(/[[:space:]]+$/, "", prefix)

                t[++count] = timer
                ac[count] = activates
                p[count] = prefix

                if (length(prefix) > wp) wp = length(prefix)
                if (length(timer) > wt) wt = length(timer)
                if (length(activates) > wa) wa = length(activates)
            }
            END {
                for (i = 1; i <= count; i++)
                    printf "%s\t%s\t%-*s  %-*s  %-*s\n", \
                        t[i], ac[i], wp, p[i], wt, t[i], wa, ac[i]
            }
        '
}

_ksys__timers_header() {

    printf '%s\n%s\n%s' \
        'Enter status · Tab select · Ctrl-R reload · Ctrl-/ preview' \
        'Ctrl-S start · Ctrl-X stop · Ctrl-N enable · Alt-D disable · Ctrl-Y cat' \
        'Ctrl-L target logs · Alt-F target follow · Alt-T trigger target now'
}

_ksys__timer_preview() {
    local scope=$1 timer=$2
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    systemctl "${scope_args[@]}" status --no-pager --full "$timer" 2>&1 |
        _ksys__preview_render text
}

# -----------------------------------------------------------------------------
# Dedicated interactive actions
# -----------------------------------------------------------------------------

# systemctl edit is an interactive editor operation. Keep it separate from the
# generic output actions and explicitly attach it to the selected unit only.
# This avoids sending an editor through a multi-selection action path.
_ksys__edit_selected() {
    local scope=$1
    local file=$2
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    [[ -r $file ]] || {
        printf 'ksys: selection file is not readable\n' >&2
        return 2
    }

    local unit count=0 line
    while IFS= read -r line; do
        [[ -n $line ]] || continue
        IFS=$'\t' read -r unit _ <<< "$line"
        [[ -n $unit ]] || continue
        count=$((count + 1))
    done < "$file"

    if ((count != 1)); then
        printf 'ksys: Ctrl-E requires exactly one selected unit.\n' >&2
        _ksys__pause
        return 1
    fi

    # Read the unit again rather than relying on the last loop variable being
    # preserved by any future refactor of the selection parser.
    while IFS= read -r line; do
        [[ -n $line ]] || continue
        IFS=$'\t' read -r unit _ <<< "$line"
        [[ -n $unit ]] && break
    done < "$file"

    # fzf owns the terminal while an execute action is running. Explicitly
    # attach the interactive editor to the real TTY so vim/nano/etc. receive
    # a terminal rather than inheriting fzf's action stdin/stdout descriptors.
    local editor_status

    if [[ $scope == user ]]; then
        systemctl --user edit "$unit" </dev/tty >/dev/tty 2>/dev/tty
        editor_status=$?
    else
        # `systemctl edit` launches the editor from the sudo-spawned process,
        # so preserve the requested shell/editor environment for the complete
        # interactive operation.
        command -v sudo >/dev/null 2>&1 || {
            printf 'ksys: sudo is required for system-manager edits.\n' >&2
            return 127
        }
        sudo -v || {
            printf 'ksys: sudo authentication failed; edit not started.\n' >&2
            return 1
        }
        sudo --preserve-env=EDITOR,MANPAGER,PAGER,PATH,TMUX,SSH_AUTH_SOCK,UNISHELL_SESSION_RUNTIME_DIR \
            systemctl edit "$unit" </dev/tty >/dev/tty 2>/dev/tty
        editor_status=$?
    fi

    return "$editor_status"
}

# systemctl cat is non-interactive, but make its invocation self-contained so
# Ctrl-Y is not affected by pager state or argument ordering.
_ksys__cat_selected() {
    local scope=$1
    local kind=$2
    local file=$3
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    [[ -r $file ]] || {
        printf 'ksys: selection file is not readable\n' >&2
        return 2
    }

    local -a units=()
    local line unit rest
    while IFS= read -r line; do
        [[ -n $line ]] || continue
        IFS=$'\t' read -r unit _ _ _ rest <<< "$line"
        [[ -n $unit ]] || continue
        units+=("$unit")
    done < "$file"

    ((${#units[@]})) || return 0

    systemctl "${scope_args[@]}" --no-pager --full cat "${units[@]}" 2>&1 |
        _ksys__render text
    _ksys__pause
}

# -----------------------------------------------------------------------------
# Generic action dispatcher
# -----------------------------------------------------------------------------

_ksys__action() {
    local action=$1
    local scope=$2
    local kind=$3
    local file=$4
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    [[ -r $file ]] || {
        printf 'ksys: selection file is not readable\n' >&2
        return 2
    }

    local -a units=() targets=()
    local line unit target rest

    if [[ $kind == timer ]]; then
        while IFS= read -r line; do
            [[ -n $line ]] || continue
            IFS=$'\t' read -r unit target rest <<< "$line"
            [[ -n $unit ]] || continue
            units+=("$unit")
            targets+=("$target")
        done < "$file"
    else
        while IFS= read -r line; do
            [[ -n $line ]] || continue
            IFS=$'\t' read -r unit _ _ _ rest <<< "$line"
            [[ -n $unit ]] || continue
            units+=("$unit")
        done < "$file"
    fi

    ((${#units[@]})) || return 0

    local i
    local -a journal_opts=()

    case $action in
        status)
            systemctl "${scope_args[@]}" status --no-pager --full "${units[@]}" |
                _ksys__render text
            _ksys__pause
            ;;

        start|stop|restart|reload|enable|disable|mask|unmask)
            _ksys__systemctl_mutate "$scope" "$action" "${units[@]}"
            ;;

        edit)
            _ksys__edit_selected "$scope" "$file"
            ;;

        cat)
            _ksys__cat_selected "$scope" "$kind" "$file"
            ;;

        logs)
            for unit in "${units[@]}"; do
                journal_opts+=( -u "$unit" )
            done
            journalctl "${scope_args[@]}" --no-pager --full -n 200 "${journal_opts[@]}" |
                _ksys__render text
            _ksys__pause
            ;;

        follow)
            for unit in "${units[@]}"; do
                journal_opts+=( -u "$unit" )
            done
            journalctl "${scope_args[@]}" -f --full -n 200 "${journal_opts[@]}"
            ;;

        previous)
            for unit in "${units[@]}"; do
                journal_opts+=( -u "$unit" )
            done
            journalctl "${scope_args[@]}" --no-pager --full -b -1 "${journal_opts[@]}" |
                _ksys__render text
            _ksys__pause
            ;;

        timer-target-logs)
            for target in "${targets[@]}"; do
                journal_opts+=( -u "$target" )
            done
            journalctl "${scope_args[@]}" --no-pager --full -n 200 "${journal_opts[@]}" |
                _ksys__render text
            _ksys__pause
            ;;

        timer-target-follow)
            ((${#targets[@]} == 1)) || {
                printf 'ksys: follow requires exactly one selected timer.\n' >&2
                _ksys__pause
                return 1
            }
            journalctl "${scope_args[@]}" -f --full -n 200 -u "${targets[0]}"
            ;;

        trigger)
            for target in "${targets[@]}"; do
                _ksys__systemctl_mutate "$scope" start "$target"
            done
            ;;

        *)
            printf 'ksys: unknown action: %s\n' "$action" >&2
            return 2
            ;;
    esac
}

# -----------------------------------------------------------------------------
# Service picker
# -----------------------------------------------------------------------------

_ksys__services() {
    local scope=${1:-system}
    _ksys__require_commands || return

    _ksys__services_list "$scope" |
        fzf \
            --with-shell='bash -c' \
            --exact \
            --multi \
            --delimiter=$'\t' \
            --with-nth=5 \
            --accept-nth='{1}' \
            --prompt="systemd $scope service ❯❯❯ " \
            --header="$(_ksys__services_header)" \
            --preview="_ksys__service_preview '$scope' {1}" \
            --preview-window='right:55%:hidden:wrap' \
            --bind="ctrl-r:reload(_ksys__services_list '$scope')" \
            --bind="enter:execute(_ksys__action status '$scope' service {+f})" \
            --bind="ctrl-s:execute(_ksys__action start '$scope' service {+f})+reload(_ksys__services_list '$scope')" \
            --bind="ctrl-x:execute(_ksys__action stop '$scope' service {+f})+reload(_ksys__services_list '$scope')" \
            --bind="alt-r:execute(_ksys__action restart '$scope' service {+f})+reload(_ksys__services_list '$scope')" \
            --bind="alt-l:execute(_ksys__action reload '$scope' service {+f})+reload(_ksys__services_list '$scope')" \
            --bind="ctrl-e:execute(_ksys__edit_selected '$scope' {+f})+reload(_ksys__services_list '$scope')" \
            --bind="ctrl-y:execute(_ksys__cat_selected '$scope' service {+f})" \
            --bind="ctrl-l:execute(_ksys__action logs '$scope' service {+f})" \
            --bind="alt-f:execute(_ksys__action follow '$scope' service {+f})" \
            --bind="alt-p:execute(_ksys__action previous '$scope' service {+f})" \
            --bind="ctrl-n:execute(_ksys__action enable '$scope' service {+f})+reload(_ksys__services_list '$scope')" \
            --bind="alt-d:execute(_ksys__action disable '$scope' service {+f})+reload(_ksys__services_list '$scope')" \
            --bind="alt-m:execute(_ksys__action mask '$scope' service {+f})+reload(_ksys__services_list '$scope')" \
            --bind="alt-u:execute(_ksys__action unmask '$scope' service {+f})+reload(_ksys__services_list '$scope')" \
            --bind='alt-enter:accept'
}

# -----------------------------------------------------------------------------
# Timer picker
# -----------------------------------------------------------------------------

_ksys__timers() {
    local scope=${1:-system}
    _ksys__require_commands || return

    _ksys__timers_list "$scope" |
        fzf \
            --with-shell='bash -c' \
            --exact \
            --multi \
            --delimiter=$'\t' \
            --with-nth=3 \
            --accept-nth='{1}' \
            --prompt="systemd $scope timer ❯❯❯ " \
            --header="$(_ksys__timers_header)" \
            --preview="_ksys__timer_preview '$scope' {1}" \
            --preview-window='right:55%:hidden:wrap' \
            --bind="ctrl-r:reload(_ksys__timers_list '$scope')" \
            --bind="enter:execute(_ksys__action status '$scope' timer {+f})" \
            --bind="ctrl-s:execute(_ksys__action start '$scope' timer {+f})+reload(_ksys__timers_list '$scope')" \
            --bind="ctrl-x:execute(_ksys__action stop '$scope' timer {+f})+reload(_ksys__timers_list '$scope')" \
            --bind="ctrl-n:execute(_ksys__action enable '$scope' timer {+f})+reload(_ksys__timers_list '$scope')" \
            --bind="alt-d:execute(_ksys__action disable '$scope' timer {+f})+reload(_ksys__timers_list '$scope')" \
            --bind="ctrl-e:execute(_ksys__edit_selected '$scope' {+f})+reload(_ksys__timers_list '$scope')" \
            --bind="ctrl-y:execute(_ksys__cat_selected '$scope' timer {+f})" \
            --bind="ctrl-l:execute(_ksys__action timer-target-logs '$scope' timer {+f})" \
            --bind="alt-f:execute(_ksys__action timer-target-follow '$scope' timer {+f})" \
            --bind="alt-t:execute(_ksys__action trigger '$scope' timer {+f})+reload(_ksys__timers_list '$scope')"
}

# -----------------------------------------------------------------------------
# Journal/unit picker
# -----------------------------------------------------------------------------

# systemctl --type=... gives us the units that are useful to select for -u
# journal filtering. We expose more than services/timers here because sockets,
# mounts, paths, targets, automounts and swaps can all be useful journal units.
_ksys__logs_list() {
    local scope=${1:-system}
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    systemctl "${scope_args[@]}" \
        list-units \
        --all \
        --no-legend \
        --no-pager \
        --full \
        --plain \
        --type=service,timer,socket,mount,target,path,automount,swap 2>/dev/null |
        awk '
            {
                unit = $1
                active = $3
                substate = $4
                if (unit == "") next

                description = ""
                if (NF >= 5) {
                    match($0, /^[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]+[^[:space:]]+[[:space:]]*/)
                    description = substr($0, RSTART + RLENGTH)
                }

                u[++count] = unit
                a[count] = active
                s[count] = substate
                d[count] = description

                if (length(unit) > wu) wu = length(unit)
                if (length(active) > wa) wa = length(active)
                if (length(substate) > ws) ws = length(substate)
            }
            END {
                for (i = 1; i <= count; i++)
                    printf "%s\t%-*s  %-*s  %-*s  %s\n", \
                        u[i], wu, u[i], wa, a[i], ws, s[i], d[i]
            }
        '
}

_ksys__logs_header() {
    printf '%s\n%s' \
        'Ctrl-L recent logs · Alt-F follow · Alt-P previous boot · Ctrl-Y cat · Alt-Y 500 lines' \
        'Ctrl-L recent logs · Alt-F follow · Alt-P previous boot · Ctrl-Y 500 lines'
}

_ksys__logs_preview() {
    local scope=$1 unit=$2
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    journalctl "${scope_args[@]}" --no-pager --full -n 40 -u "$unit" 2>&1 |
        _ksys__preview_render text
}

_ksys__logs_snapshot_selected() {
    local scope=$1
    local file=$2
    local -a scope_args=()
    [[ $scope == user ]] && scope_args+=(--user)

    [[ -r $file ]] || {
        printf 'ksys: selection file is not readable\n' >&2
        return 2
    }

    local -a units=()
    local line unit rest
    while IFS= read -r line; do
        [[ -n $line ]] || continue
        IFS=$'\t' read -r unit _ _ _ rest <<< "$line"
        [[ -n $unit ]] || continue
        units+=("$unit")
    done < "$file"

    ((${#units[@]})) || return 0

    local -a journal_opts=()
    for unit in "${units[@]}"; do
        journal_opts+=( -u "$unit" )
    done

    journalctl "${scope_args[@]}" --no-pager --full -n 500 "${journal_opts[@]}" 2>&1 |
        _ksys__render text
    _ksys__pause
}

_ksys__logs() {
    local scope=${1:-system}
    _ksys__require_commands || return

    _ksys__logs_list "$scope" |
        fzf \
            --with-shell='bash -c' \
            --exact \
            --multi \
            --delimiter=$'\t' \
            --with-nth=2 \
            --accept-nth='{1}' \
            --prompt="systemd $scope logs ❯❯❯ " \
            --header="$(_ksys__logs_header)" \
            --preview="_ksys__logs_preview '$scope' {1}" \
            --preview-window='right:55%:hidden:wrap' \
            --bind="ctrl-r:reload(_ksys__logs_list '$scope')" \
            --bind="enter:execute(_ksys__action logs '$scope' service {+f})" \
            --bind="ctrl-l:execute(_ksys__action logs '$scope' service {+f})" \
            --bind="alt-f:execute(_ksys__action follow '$scope' service {+f})" \
            --bind="alt-p:execute(_ksys__action previous '$scope' service {+f})" \
            --bind="ctrl-e:execute(_ksys__edit_selected '$scope' {+f})+reload(_ksys__logs_list '$scope')" \
            --bind="alt-y:execute(_ksys__logs_snapshot_selected '$scope' {+f})" \
            --bind="ctrl-y:execute(_ksys__cat_selected '$scope' service {+f})" \
            --bind='alt-enter:accept'
}

# -----------------------------------------------------------------------------
# Top-level menu
# -----------------------------------------------------------------------------

_ksys__menu_list() {
    # IMPORTANT: use printf with a real tab. Do not put a literal "\\t" inside
    # single quotes, which would render the backslash+t artifact in fzf.
    printf '%s\t%s\n' \
        system-services 'System · services' \
        system-timers   'System · timers' \
        system-logs     'System · journal logs' \
        user-services   'User · services' \
        user-timers     'User · timers' \
        user-logs       'User · journal logs'
}

_ksys__menu_header() {
    printf '%s\n%s' \
        'Enter select · Ctrl-R reload · Ctrl-/ preview · Esc exit' \
        'Exact matching'
}

_ksys__menu_preview() {
    local selection=$1

    case $selection in
        system-services)
            systemctl --no-pager --full --plain status systemd-logind.service 2>&1
            ;;
        system-timers)
            systemctl --no-pager --full --plain list-timers --all 2>&1
            ;;
        system-logs)
            journalctl --no-pager --full -n 25 2>&1
            ;;
        user-services)
            systemctl --user --no-pager --full --plain list-units --type=service --all 2>&1
            ;;
        user-timers)
            systemctl --user --no-pager --full --plain list-timers --all 2>&1
            ;;
        user-logs)
            journalctl --user --no-pager --full -n 25 2>&1
            ;;
    esac | _ksys__preview_render text
}

_ksys__menu() {
    local selected

    selected=$(
        _ksys__menu_list |
            fzf \
                --with-shell='bash -c' \
                --exact \
                --no-multi \
                --delimiter=$'\t' \
                --with-nth=2 \
                --accept-nth='{1}' \
                --prompt='systemd ❯❯❯ ' \
                --header="$(_ksys__menu_header)" \
                --preview='_ksys__menu_preview {1}' \
                --preview-window='right:55%:hidden:wrap' \
                --bind='ctrl-r:reload(_ksys__menu_list)'
    ) || return 0

    case $selected in
        system-services) _ksys__services system ;;
        system-timers) _ksys__timers system ;;
        system-logs) _ksys__logs system ;;
        user-services) _ksys__services user ;;
        user-timers) _ksys__timers user ;;
        user-logs) _ksys__logs user ;;
    esac
}

# Scope-only menu used by `ksys user`, `ksys -u`, and `ksys system`.
_ksys__scope_menu_list() {
    local scope=$1
    local label
    label=$([[ $scope == user ]] && printf 'User' || printf 'System')

    printf '%s\t%s\n' \
        services "$label · services" \
        timers   "$label · timers" \
        logs     "$label · journal logs"
}

_ksys__scope_menu() {
    local scope=${1:-system}
    local selected

    selected=$(
        _ksys__scope_menu_list "$scope" |
            fzf \
                --with-shell='bash -c' \
                --exact \
                --no-multi \
                --delimiter=$'\t' \
                --with-nth=2 \
                --accept-nth='{1}' \
                --prompt="systemd $scope ❯❯❯ " \
                --header="$(_ksys__menu_header)" \
                --preview="_ksys__menu_preview ${scope}-{1}" \
                --preview-window='right:55%:hidden:wrap' \
                --bind="ctrl-r:reload(_ksys__scope_menu_list '$scope')"
    ) || return 0

    case $selected in
        services) _ksys__services "$scope" ;;
        timers) _ksys__timers "$scope" ;;
        logs) _ksys__logs "$scope" ;;
    esac
}

# -----------------------------------------------------------------------------
# Public command
# -----------------------------------------------------------------------------

_ksys__help() {
    cat <<'EOF'
ksys - fzf-driven systemd workflows

Usage:
  ksys                         Choose system/user services, timers, or logs.
  ksys services               Browse system services.
  ksys timers                 Browse system timers.
  ksys logs                   Browse system journal by unit.
  ksys user services          Browse user services.
  ksys user timers            Browse user timers.
  ksys user logs              Browse user journal by unit.
  ksys --user services        Same as `ksys user services`.
  ksys -u timers              Same as `ksys user timers`.

Matching and preview:
  All pickers use --exact by default.
  Previews are hidden initially; use Ctrl-/ to show them.
  bat is preferred for text rendering; cat is the fallback.
  System-manager mutations use sudo and preserve:
    EDITOR, MANPAGER, PAGER, PATH, TMUX, SSH_AUTH_SOCK, UNISHELL_SESSION_RUNTIME_DIR
  User-manager operations do not use sudo.

Service keys:
  Enter       status
  Tab         multi-select
  Ctrl-R      reload
  Ctrl-/      toggle preview
  Ctrl-S      start
  Ctrl-X      stop
  Alt-R       restart
  Alt-L       reload
  Ctrl-E      edit unit/drop-in
  Ctrl-Y      cat unit/drop-ins
  Ctrl-L      recent logs
  Alt-F       follow logs
  Alt-P       previous-boot logs
  Ctrl-N      enable
  Alt-D       disable
  Alt-M       mask
  Alt-U       unmask
  Alt-Enter   accept selected unit names and exit

Timer keys:
  Enter       status
  Tab         multi-select
  Ctrl-R      reload
  Ctrl-/      toggle preview
  Ctrl-S      start timer
  Ctrl-X      stop timer
  Ctrl-N      enable
  Alt-D       disable
  Ctrl-E      edit timer unit/drop-in
  Ctrl-Y      cat timer unit/drop-ins
  Ctrl-L      logs of activated service(s)
  Alt-F       follow activated service (one timer)
  Alt-T       trigger activated service(s) now

Log keys:
  Enter       recent logs
  Ctrl-L      recent logs
  Alt-F       follow logs
  Alt-P       previous-boot logs
  Ctrl-E      edit selected unit/drop-in
  Ctrl-Y      cat selected unit/drop-in
EOF
}

ksys() {
    local scope=system
    local command=''
    local explicit_scope=false

    case ${1-} in
        -u|--user)
            scope=user
            explicit_scope=true
            shift
            ;;
    esac

    if [[ ${1-} == user || ${1-} == system ]]; then
        scope=$1
        explicit_scope=true
        shift
    fi

    command=${1-}

    case $command in
        service|services|svc)
            shift
            _ksys__services "$scope" "$@"
            ;;
        timer|timers)
            shift
            _ksys__timers "$scope" "$@"
            ;;
        log|logs|journal)
            shift
            _ksys__logs "$scope" "$@"
            ;;
        help|-h|--help)
            _ksys__help
            ;;
        '')
            if [[ $explicit_scope == true ]]; then
                _ksys__scope_menu "$scope"
            else
                _ksys__menu
            fi
            ;;
        *)
            printf 'ksys: unknown command: %s\n' "$command" >&2
            printf 'Run `ksys --help` for usage.\n' >&2
            return 2
            ;;
    esac
}

# fzf execute/preview/reload commands run in child Bash processes because all
# pickers explicitly use --with-shell=bash -c. Export the helpers those child
# processes need.
export -f \
    _ksys__require_commands \
    _ksys__pause \
    _ksys__render \
    _ksys__preview_render \
    _ksys__clip \
    _ksys__scope_args \
    _ksys__scope_label \
    _ksys__sudo_systemctl \
    _ksys__systemctl_mutate \
    _ksys__services_list \
    _ksys__services_header \
    _ksys__service_preview \
    _ksys__timers_list \
    _ksys__timers_header \
    _ksys__timer_preview \
    _ksys__edit_selected \
    _ksys__cat_selected \
    _ksys__logs_snapshot_selected \
    _ksys__action \
    _ksys__services \
    _ksys__timers \
    _ksys__logs_list \
    _ksys__logs_header \
    _ksys__logs_preview \
    _ksys__logs \
    _ksys__menu_list \
    _ksys__menu_header \
    _ksys__menu_preview \
    _ksys__scope_menu_list \
    _ksys__scope_menu \
    _ksys__menu \
    _ksys__help \
    ksys

