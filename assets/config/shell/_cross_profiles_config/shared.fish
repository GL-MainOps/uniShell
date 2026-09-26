# ╔═══════════════════════════════════════════════════════════════
# ║ SHELL OPTS

# ╭────────────────────────────────╮
# │ SAME BASH (!!) BEHAVIOR        │
# ╰────────────────────────────────╯
function last_history_item; echo $history[1]; end; abbr -a !! --position anywhere --function last_history_item

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ VARIABLES

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ COMPLETIONS

# ╭────────────────────────────────╮
# │ STATIC COMPLETION FILES        │
# ╰────────────────────────────────╯
set -a fish_complete_path "$UNISHELL_CONFIG_SHELL_PATH/_completions" \
    (fd --hidden --type d . "$UNISHELL_CONFIG_SHELL_PATH/_completions")

# ╭────────────────────────────────╮
# │ ON-DEMAND COMPLETION SOURCING  │
# ╰────────────────────────────────╯
type -q bat; and bat --completion fish | source
type -q crictl; and crictl completion fish | source
type -q fd; and fd --gen-completions fish | source
type -q helm; and helm completion fish | source
type -q kubectl; and kubectl completion fish | source
# type -q kubeadm; and kubeadm completion fish | source
type -q rg; and rg --generate complete-fish | source
# type -q zellij; and zellij setup --generate-completion fish | source

# docker needs a version gate (completion subcommand requires >=23.0)
if type -q docker
    set -l docker_version (docker version --format '{{.Client.Version}}' 2>/dev/null)

    if test -n "$docker_version"
        set -l docker_version_parts (string split '.' -- "$docker_version")
        set -l docker_major $docker_version_parts[1]
        set -l docker_minor $docker_version_parts[2]

        if test "$docker_major" -gt 23 \
            -o \( "$docker_major" -eq 23 -a "$docker_minor" -ge 0 \)
            docker completion fish | source
        end
    end
end

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ FUNCTIONS

set -a fish_function_path "$UNISHELL_CONFIG_SHELL_PATH/_functions" \
    (fd --hidden --type d . "$UNISHELL_CONFIG_SHELL_PATH/_functions")

# ╭────────────────────────────────╮
# │ QUICK FUNCTIONS                │
# ╰────────────────────────────────╯
function mkcd
    mkdir -p -- $argv[1] && cd -- $argv[1]
end

function clhist
    history clear
end

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ KEYBINDS

# HINT: running `fish_key_reader` helps you find key names

for file in (fd --hidden --type f --extension fish . "$UNISHELL_CONFIG_SHELL_PATH/_keybinds")
    source "$file"
end

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ PROGRAMS-RELATED CONFIGURATION

# ╭────────────────────────────────╮
# │ SYSTEM                         │
# ╰────────────────────────────────╯
# ┌────────────────┐
# │ SUDO           │
# └────────────────┘
set -l unishell_envs (set --names | string match 'UNISHELL_*' | string join ',')
set -gx SUDO_PRESERVED_VARIABLES \
    "$SUDO_INITIAL_PRESERVED_VARIABLES,$unishell_envs"
function sudo
    command sudo --preserve-env="$SUDO_PRESERVED_VARIABLES" $argv
end

# ┌────────────────┐
# │ VIM            │
# └────────────────┘
function vim
    command vim -c "source $vimrc" $argv
end

alias v vim
alias vi vim

function svim
    sudo vim -c "source $vimrc" $argv
end

alias sv svim

# ╭────────────────────────────────╮
# │ OTHER                          │
# ╰────────────────────────────────╯

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ EVALs

# ╭────────────────────────────────╮
# │ FZF                            │
# ╰────────────────────────────────╯
fzf --fish | source

# ╭────────────────────────────────╮
# │ ZOXIDE                         │
# ╰────────────────────────────────╯
zoxide init fish | source

# ╚═══════════════════════════════════════════════════════════════
# vim: set ft=fish:
