# ╔═══════════════════════════════════════════════════════════════
# ║ FZF FUNCTIONS

# ╭────────────────────────────────╮
# │ BROWSE HISTORY ON MOD+UP       │
# ╰────────────────────────────────╯
function fzf-history
    set -l query (commandline)
    set -l selected (history | command fzf --tac --no-sort --exact --query="$query" --border-label='SHELL HISTORY' --preview 'printf "%s\\n" {}' --preview-window='down:25%:wrap' --layout=default | string collect)
    test -n "$selected"; and commandline --replace -- "$selected"
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
bind shift-up fzf-history
bind ctrl-up fzf-history
bind ctrl-shift-up fzf-history

# ╭────────────────────────────────╮
# │ ZOXIDE ON CDABLE VARS          │
# ╰────────────────────────────────╯
function fz
    if not type -q zoxide
        printf '%s\n' 'ze: zoxide not found' >&2
        return 1
    end

    set -l sel (
        for var in (set --names)
            for val in $$var
                test -d "$val"; or continue
                printf '%s\t%s=%s\n' "$val" "$var" "$val"
            end
        end |
        fzf --border-label='Zoxide on Vars' \
            --exact \
            --delimiter=(printf '\t') \
            --with-nth=2 \
            --header='Enter: zoxide add + cd' \
            --preview='eza --group-directories-first -AMlF --no-permissions --no-user --no-time --total-size --tree --level=2 --color=always --icons=auto {1}' \
            --preview-window='right:70%:wrap'
    )

    test -n "$sel"; or return

    set -l val (string split -m1 (printf '\t') -- "$sel")[1]

    zoxide add --score 4 -- "$val"
    and builtin cd -- "$val"
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
bind alt-Z 'fz'

# ╭────────────────────────────────╮
# │ ENVIRONMENT VARIABLES BROWSER  │
# ╰────────────────────────────────╯
function fenv
    env | command fzf --border-label='Environment Variables' --exact --preview 'printf "%s\\n" {}' --preview-window='down:25%:wrap'
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
bind alt-e fenv

# ╭────────────────────────────────╮
# │ PROCESS KILLER                 │
# ╰────────────────────────────────╯
function fkill
    set -l signal 9
    if test (count $argv) -gt 0
        set signal $argv[1]
    end
    ps -ef | sed 1d | command fzf --border-label='Kill a Running Processes' --multi --preview='' | awk '{print $2}' | xargs -r kill -$signal
end

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘

# ╭────────────────────────────────╮
# │ VIM ANYWHERE                   │
# ╰────────────────────────────────╯
function fvim
    set -l files_command 'fd --type f --hidden --exclude .git'
    set -l fzf_options ''

    set -q FZF_CTRL_T_COMMAND; and set files_command $FZF_CTRL_T_COMMAND
    set -q FZF_CTRL_T_OPTS; and set fzf_options $FZF_CTRL_T_OPTS

    set -l files (
        eval $files_command |
        eval "fzf --border-label='VIM Anywhere' --multi --print0 $fzf_options" |
        string split0
    )

    test (count $files) -gt 0; and command vim -- $files
end

alias vv='fvim'

# ┌────────────────┐
# │ KEYBINDINGS    │
# └────────────────┘
bind alt-v 'fvim'

# ╭────────────────────────────────╮
# │ GREP TO VIM                    │
# ╰────────────────────────────────╯
function frg
    set -l query ''
    if test (count $argv) -gt 0
        set query $argv[1]
    end
    set -l selected (rg --line-number --no-heading --color=always -- $query | command fzf --border-label='RIPGREP TO VIM' --multi --delimiter ':' --preview 'bat --color=always {1} --highlight-line {2}' | cut -d: -f1-2 | string collect)
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

# ╚═══════════════════════════════════════════════════════════════
# vim: set ft=fish:
