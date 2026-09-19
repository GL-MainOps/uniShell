# SIMPLE EXAMPLE WHERE DEFAULT_OPTS ARE ALREADY SET
# kpod() {
#     kubectl get pods -A --no-headers |
#         fzf \
#             --header='ENTER logs | CTRL-D describe | CTRL-R reload' \
#             --preview 'kubectl describe pod -n {1} {2}' \
#             --bind 'ctrl-r:reload(kubectl get pods -A --no-headers)' \
#             --bind 'ctrl-d:execute(kubectl describe pod -n {1} {2})' \
#             --bind 'enter:execute(kubectl logs -n {1} {2})'
# }

# BROWSE HISTORY ON UP
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

bind -x '"\e[1;5A":fzf-history'
bind -x '"\e[1;2A":fzf-history'


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

# Environment variable inspector
fenv() {
    env | fzf --exact --preview 'echo {}' --preview-window='down:25%:wrap'
}
bind -x '"\ee": fenv;'


# Process killer
fkill() {
    local pid
    pid=$(ps -ef | sed 1d | fzf -m --preview='' | awk '{print $2}') && echo "$pid" | xargs -r kill -"${1:-9}"
}
bind -x '"\ep": fkill;'


# vim Anywhere
vv() {
    local -a files

    mapfile -t files < <(
        eval "$FZF_CTRL_T_COMMAND" |
            eval "fzf $FZF_CTRL_T_OPTS"
    )

    ((${#files[@]})) && vim "${files[@]}"
}
bind -x '"\ev": vv;'

# GREP TO VIM
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
