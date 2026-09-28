# ╔═══════════════════════════════════════════════════════════════
# ║ SHELL PRE-INIT STEPS

set -g fish_history fish
fish_config theme choose 'fish default'

# ╚═══════════════════════════════════════════════════════════════



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
for file in (fd --hidden --type f --extension fish --exclude "kubectl-completion.fish" . "$UNISHELL_CONFIG_SHELL_PATH/_completions")
    source "$file"
end

# Auto loading kubectl completions instead of sourcing at every launch
set -a fish_complete_path "$UNISHELL_CONFIG_SHELL_PATH/_completions/kubectl"

# ╭────────────────────────────────╮
# │ ON-DEMAND COMPLETION SOURCING  │
# ╰────────────────────────────────╯
# ┌────────────────┐
# │ FOR-REFERENCE  │
# └────────────────┘
# type -q bat; and bat --completion fish | source
# type -q fd; and fd --gen-completions fish | source
# type -q helm; and helm completion fish | source
# type -q kubectl; and kubectl completion fish | source
# type -q rg; and rg --generate complete-fish | source
# type -q zellij; and zellij setup --generate-completion fish | source

# ┌────────────────┐
# │ DYNAMIC SOURCE │
# └────────────────┘
# type -q crictl; and crictl completion fish | source
# type -q kubeadm; and kubeadm completion fish | source

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ FUNCTIONS
# for file in (fd --hidden --type f --extension fish . "$UNISHELL_CONFIG_SHELL_PATH/_functions")
#     source "$file"
# end

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
# │ STARTSHIP      │
# └────────────────┘
function starship_transient_prompt_func
  echo -n ""; starship module sudo; echo -n ""
  starship module character
  starship module line_break
end

function starship_transient_rprompt_func
  echo -n ""; starship module custom.long-timestamp; echo -n ""

end


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
    sudo --preserve-env="$SUDO_PRESERVED_VARIABLES" vim -c "source $vimrc" $argv
end

alias sv svim

# ╭────────────────────────────────╮
# │ OTHER                          │
# ╰────────────────────────────────╯

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ EVALs

# ╭────────────────────────────────╮
# │ STARSHIP                       │
# ╰────────────────────────────────╯
starship init fish | source && enable_transience

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
