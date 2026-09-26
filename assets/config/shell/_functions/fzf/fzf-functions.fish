# ╔═══════════════════════════════════════════════════════════════
# ║ FZF FUNCTIONS

# ╭────────────────────────────────╮
# │ BROWSE HISTORY ON SHIFT/CTRL UP│
# ╰────────────────────────────────╯
function fzf-history
    set -l query (commandline)
    set -l selected (history | command fzf --tac --no-sort --exact --query="$query" --border-label 'SHELL HISTORY' --preview 'printf "%s\\n" {}' --preview-window='down:25%:wrap' --layout=default | string collect)
    test -n "$selected"; and commandline --replace -- "$selected"
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# bind '\\e\\[1;5A' fzf-history
# bind '\\e\\[1;2A' fzf-history

# ╭────────────────────────────────╮
# │ ZOXIDE ON CDABLE VARS          │
# ╰────────────────────────────────╯
function ze
    command -sq zoxide; or begin
        printf 'ze: zoxide not found\\n' >&2
        return 127
    end
    set -l rows
    for name in (set --names)
        for value in $$name
            if test -n "$value"; and test -d "$value"
                set -a rows "$value\t$name=$value"
            end
        end
    end
    set -l selected (printf '%s\\n' $rows | command fzf --exact --delimiter='\\t' --with-nth=2 --header='Enter: zoxide add + cd' --preview='eza --group-directories-first -AMlF --no-permissions --no-user --no-time --total-size --tree --level=2 --color=always --icons=auto {1}' --preview-window='right:70%:wrap' | string collect)
    test -n "$selected"; or return 0
    set -l path (string split -m 1 \\t -- $selected)[1]
    command zoxide add --score 4 -- $path; and cd -- $path
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# N/A

# ╭────────────────────────────────╮
# │ ENVIRONMENT VARIABLES BROWSER  │
# ╰────────────────────────────────╯
function fenv
    env | command fzf --exact --preview 'printf "%s\\n" {}' --preview-window='down:25%:wrap'
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# bind -x '"\ee": fenv;'


# ╭────────────────────────────────╮
# │ PROCESS KILLER                 │
# ╰────────────────────────────────╯
function fkill
    set -l signal 9
    if test (count $argv) -gt 0
        set signal $argv[1]
    end
    ps -ef | sed 1d | command fzf --multi --preview='' | awk '{print $2}' | xargs -r kill -$signal
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
bind -x '"\ep": fkill;'


# ╭────────────────────────────────╮
# │ VIM ANYWHERE                   │
# ╰────────────────────────────────╯
function vv
    set -l files_command 'fd --type f --hidden --exclude .git'
    set -l fzf_options ''
    set -q FZF_CTRL_T_COMMAND; and set files_command $FZF_CTRL_T_COMMAND
    set -q FZF_CTRL_T_OPTS; and set fzf_options $FZF_CTRL_T_OPTS
    set -l selected (eval "$files_command | fzf $fzf_options" | string collect)
    test -n "$selected"; and command vim (string split \\n -- $selected)
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# bind -x '"\ev": vv;'


# ╭────────────────────────────────╮
# │ GREP TO VIM                    │
# ╰────────────────────────────────╯
function fv
    set -l query ''
    if test (count $argv) -gt 0
        set query $argv[1]
    end
    set -l selected (rg --line-number --no-heading --color=never -- $query | command fzf --multi --delimiter ':' --preview 'bat --color=always {1} --highlight-line {2}' | cut -d: -f1-2 | string collect)
    test -n "$selected"; or return 0
    set -l vim_args
    for item in (string split \\n -- $selected)
        set -l fields (string split -m 1 ':' -- $item)
        set -a vim_args "+$fields[2]" $fields[1]
    end
    command vim $vim_args
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
# N/A

# ╚═══════════════════════════════════════════════════════════════
# vim: set ft=fish:
