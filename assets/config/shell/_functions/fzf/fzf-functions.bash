# ╔═══════════════════════════════════════════════════════════════
# ║ FZF FUNCTIONS

# ╭────────────────────────────────╮
# │ BROWSE HISTORY ON SHIFT/CTRL UP│
# ╰────────────────────────────────╯
fzf-history() {
    local -a selected

    mapfile -d '' -t selected < <(
        fc -rl 1 |
            awk '
                /^[[:space:]]*[0-9]+[[:space:]]/ {
                    if (entry != "")
                        printf "%s%c", entry, 0

                    entry = $0
                    sub(/^[[:space:]]*[0-9]+[[:space:]]*/, "", entry)
                    next
                }

                {
                    entry = entry "\n" $0
                }

                END {
                    if (entry != "")
                        printf "%s%c", entry, 0
                }
            ' |
            fzf \
                --read0 \
                --print0 \
                --no-sort \
                --exact \
                --query="$READLINE_LINE" \
                --border-label 'SHELL HISTORY' \
                --preview 'echo {}' \
                --preview-window='down:25%:wrap' \
                --layout=default
    )

    ((${#selected[@]})) || return

    READLINE_LINE=${selected[0]}
    READLINE_POINT=${#READLINE_LINE}
}

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# bind -x '"\e[1;5A":fzf-history'
# bind -x '"\e[1;2A":fzf-history'


# ╭────────────────────────────────╮
# │ ZOXIDE ON CDABLE VARS          │
# ╰────────────────────────────────╯
ze() {
    command -v zoxide >/dev/null 2>&1 || { printf 'fcdvar: zoxide not found\n' >&2; return 1; }

    local sel var val

    sel=$(
        while read -r var; do
            val=${!var-}
            [[ -n $val && -d $val ]] && printf '%s\t%s=%s\n' "$val" "$var" "$val"
        done < <(compgen -v) |
        fzf --exact \
            --delimiter=$'\t' \
            --with-nth=2 \
            --header='Enter: zoxide add + cd' \
            --preview='eza --group-directories-first -AMlF --no-permissions --no-user --no-time --total-size --tree --level=2 --color=always --icons=auto {1}' \
            --preview-window='right:70%:wrap'
    ) || return

    [[ -n $sel ]] || return
    val=${sel%%$'\t'*}

    zoxide add --score 4 -- "$val" && builtin cd -- "$val"
}

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# N/A


# ╭────────────────────────────────╮
# │ ENVIRONMENT VARIABLES BROWSER  │
# ╰────────────────────────────────╯
fenv() {
    env | fzf --exact --preview 'echo {}' --preview-window='down:25%:wrap'
}

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# bind -x '"\ee": fenv;'


# ╭────────────────────────────────╮
# │ PROCESS KILLER                 │
# ╰────────────────────────────────╯
fkill() {
    local pid
    pid=$(ps -ef | sed 1d | fzf -m --preview='' | awk '{print $2}') && echo "$pid" | xargs -r kill -"${1:-9}"
}

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# bind -x '"\ep": fkill;'


# ╭────────────────────────────────╮
# │ VIM ANYWHERE                   │
# ╰────────────────────────────────╯
vv() {
    local -a files

    mapfile -t files < <(
        eval "$FZF_CTRL_T_COMMAND" |
            eval "fzf $FZF_CTRL_T_OPTS"
    )

    ((${#files[@]})) && vim "${files[@]}"
}

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# bind -x '"\ev": vv;'


# ╭────────────────────────────────╮
# │ GREP TO VIM                    │
# ╰────────────────────────────────╯
fv() {
    local -a selected vim_args
    local item file line

    mapfile -t selected < <(
        rg --line-number --no-heading --color=always "${1:-}" |
            fzf --ansi \
                --multi \
                --delimiter : \
                --preview 'bat --color=always {1} --highlight-line {2}' |
            cut -d: -f1-2
    ) || return

    for item in "${selected[@]}"; do
        file=${item%%:*}
        line=${item#*:}

        vim_args+=( "+${line%%:*}" "$file" )
    done

    ((${#vim_args[@]})) && vim "${vim_args[@]}"
}

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# N/A

# ╚═══════════════════════════════════════════════════════════════
# vim: set ft=bash:
