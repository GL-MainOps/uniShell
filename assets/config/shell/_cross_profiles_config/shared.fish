# ╔═══════════════════════════════════════════════════════════════
# ║ SHELL PRE-INIT STEPS

# ╭────────────────────────────────╮
# │ SPECIAL COLORING               │
# ╰────────────────────────────────╯

set -g fish_history fish
fish_config theme choose 'fish default'

# Design:
#
#   Command       → bold
#   Builtin       → bright green
#   Function      → bright cyan
#   Keyword       → bright magenta
#   Option        → bright yellow
#   Parameter     → cyan
#   Quote         → yellow
#   Path          → underline
#   Redirection   → bright blue
#   Error         → bright red
#
# Completion pager:
#
#   Completion    → bright/light text
#   Description   → bright yellow
#   Selection     → dark background + bright text
#
# Designed for a DARK terminal background.
#
# Fish documentation:
# https://fishshell.com/docs/current/interactive.html

# ┌────────────────┐
# │ BASE           │
# └────────────────┘

# Default command-line text.
set -g fish_color_normal normal

# ┌────────────────┐
# │ COMMANDS       │
# └────────────────┘

# External commands.
#
#   rsync -a ./src ./dst
#   ^^^^^
set -g fish_color_command --bold

# Fish builtins.
#
# Examples:
#   cd
#   set
#   string
#   math
#   command
#   type
set -g fish_color_builtin brgreen

# User-defined / autoloaded functions.
#
#   my_function --foo
#   ^^^^^^^^^^^
set -g fish_color_function brcyan

# Fish language keywords.
#
# Examples:
#   if
#   else
#   end
#   for
#   while
#   function
set -g fish_color_keyword brmagenta --bold

# ┌────────────────┐
# │ ARGS / OPTS    │
# └────────────────┘

# Normal command arguments.
#
#   rsync -a ./bin/unishell-slim ./
#         ^^^^^^^^^^^^^^^^^^^^^^^^^^^
set -g fish_color_param cyan

# Command options / flags.
#
# Examples:
#   -aPtvuhr
#   --archive
#   --progress
set -g fish_color_option bryellow

# Quoted strings.
set -g fish_color_quote yellow

# Existing filesystem paths.

# Only adding --underline means the path keeps its normal parameter color.
set -g fish_color_valid_path --underline

# ┌────────────────┐
# │ OPERATORS      │
# └────────────────┘

# Shell operators.
set -g fish_color_operator bryellow

# Command terminators / separators.
set -g fish_color_end brmagenta

# Input/output redirections.
#
# Examples:
#   >/dev/null
#   2>&1
#   <input.txt
set -g fish_color_redirection brblue

# Escape sequences.
set -g fish_color_escape bryellow --bold

# Comments.
set -g fish_color_comment red

# ┌────────────────┐
# │ ERRORS / STATUS│
# └────────────────┘

# Invalid syntax / parser errors.
set -g fish_color_error brred --bold

# Non-zero command status.
set -g fish_color_status brred

# Cancelled command line / Ctrl-C.
set -g fish_color_cancel -r


# ┌────────────────┐
# │ PROMPT         │
# └────────────────┘

# Current working directory.
set -g fish_color_cwd green

# Current working directory when running as root.
set -g fish_color_cwd_root red

# Username.
set -g fish_color_user brgreen

# Local hostname.
set -g fish_color_host normal

# Remote hostname.
set -g fish_color_host_remote brcyan


# ┌────────────────┐
# │ AUTOSUGGESTIONS│
# └────────────────┘

# Muted, but still visible.
set -g fish_color_autosuggestion 666

# Current history entry.
set -g fish_color_history_current --bold

# Search matches.

# Bright yellow on a dark background.
set -g fish_color_search_match bryellow --background=brblack

# Selected text.

# Dark background + light text works well with a dark terminal.
set -g fish_color_selection white --bold --background=brblack

# Matching brackets / delimiters.
set -g fish_color_match --background=brblue

# ┌────────────────┐
# │ COMP. PAGER    │
# └────────────────┘

# Normal:
#
#   prefix       → bright red + bold
#   completion   → white
#   description  → bright yellow + bold
#
# Selected:
#
#   background   → dark blue
#   text         → white / bright red / yellow
#
# This is intended for a DARK terminal background.


# ── Normal completion rows ──────────────────────────────────────────────────

# Part matching what you typed.
#
# Example:
#
#   kubectl get po<TAB>
#                 ^^
set -g fish_pager_color_prefix brred --bold

# Actual completion candidate.

set -g fish_pager_color_completion cyan --bold

# Completion description.

# Requested bright yellow.
set -g fish_pager_color_description yellow --bold


# ── Selected completion ─────────────────────────────────────────────────────

# Selected row background.

# Use a DARK blue rather than a bright blue so light foreground text remains
# clearly readable.
set -g fish_pager_color_selected_background --background=blue

# Selected prefix.
set -g fish_pager_color_selected_prefix white --bold

# Selected completion.
set -g fish_pager_color_selected_completion white --bold

# Selected description.
set -g fish_pager_color_selected_description black


# ── Secondary / alternating rows ────────────────────────────────────────────

# Slightly different dark background for secondary rows.
set -g fish_pager_color_secondary_background normal

# Secondary completion text.
set -g fish_pager_color_secondary_completion cyan --bold

# Secondary descriptions.
set -g fish_pager_color_secondary_description yellow --bold


# ── Pager progress ──────────────────────────────────────────────────────────

# Pager progress indicator.
set -g fish_pager_color_progress white --bold

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
for file in (fd --hidden --type f --extension fish . "$UNISHELL_CONFIG_SHELL_PATH/_functions")
    source "$file"
end

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
