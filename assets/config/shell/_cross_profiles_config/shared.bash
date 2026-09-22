################################################
## SHELL PRE-INIT STEPS
################################################
### SKIP IF NOT INTERACTIVE
case $- in
    *i*) ;;
    *) return;;
esac

# ======================================

### INITIAL SOURCING
if [ -f /etc/bashrc ]; then
    . /etc/bashrc
fi
################################################



################################################
## VARIABLES
################################################

# EDITORS & PAGERS
declare -A TERMVIEWERS=(
    [bat]="export GROFF_NO_SGR=1 MANPAGER=\"sh -c 'col -bx | bat -l man -p'\" PAGER=\"bat -p --paging=always\""
    [vim]="export EDITOR=vim"
    # [diffprog]="export EDITOR=vim"
)
for cmd in "${!TERMVIEWERS[@]}"; do
    command -v "$cmd" >/dev/null 2>&1 && eval "${TERMVIEWERS[$cmd]}"
done

# `sudo -E` ALTERMATIVE
UNISHELL_ENVS=$(IFS=,; echo "${!UNISHELL_*}")
export SUDO_PRESERVED_VARIABLES="$SUDO_INITIAL_PRESERVED_VARIABLES,$UNISHELL_ENVS"
################################################



################################################
## EVALS & COMPLETIONS
################################################

# HOST NATIVE COMPLETIONS
for bc in \
    /usr/share/bash-completion/bash_completion \
    /etc/bash_completion \
    /usr/local/etc/profile.d/bash_completion.sh \
    /opt/homebrew/etc/profile.d/bash_completion.sh
do
    if [[ -r "$bc" ]]; then
        source "$bc"
        break
    fi
done

# STATIC COMPLETION FILES
for file in $(fd -e bash -t f . "${UNISHELL_CONFIG_SHELL_PATH}/_completions"); do
    source "$file"
done


# ON-DEMAND COMPLETION SOURCING
declare -A completions=(
    [bat]="bat --completion bash"
    [crictl]="crictl completion bash"
    [fd]="fd --gen-completions bash"
    [helm]="helm completion bash"
    [kubectl]="kubectl completion bash"
    [kubeadm]="kubeadm completion bash"
    [rg]="rg --generate complete-bash"
    [zellij]="zellij setup --generate-completion bash"
)
for cmd in "${!completions[@]}"; do
    command -v "$cmd" >/dev/null 2>&1 && source <(${completions[$cmd]})
done


# docker needs a version gate (completion subcommand requires >=23.0)
if command -v docker >/dev/null 2>&1; then
    docker_ver=$(docker version --format '{{.Client.Version}}' 2>/dev/null)
    if [[ -n "$docker_ver" ]] && printf '%s\n' "23.0.0" "$docker_ver" | sort -V -C; then
        source <(docker completion bash)
    fi
fi

# CUSTOM K8S ALIAS
alias k="kubectl"
complete -F __start_kubectl k

# RELATED SHELL COMPLETIONS OPTIONS
# On first <TAB> with multiple matches, list them immediately instead of requiring a second <TAB> press (default readline behavior needs two presses: one to beep, one to list).
bind 'set show-all-if-ambiguous on'
# Same immediate-listing behavior, but specifically for the case where completions share a common prefix that's already fully typed (so nothing would be inserted) — without this, that case still needs a second <TAB>.
bind 'set show-all-if-unmodified on'
# Makes filename/command completion case-insensitive, e.g. typing doc<TAB> matches Documents/.
bind 'set completion-ignore-case on'
# Allow programmable completion to work with aliases
shopt -s progcomp_alias

# FZF
eval "$(fzf --bash)"

# ZOXIDE
eval "$(zoxide init bash)"
################################################



################################################
## FUNCTIONS
################################################
for file in $(fd -e bash -t f . "${UNISHELL_CONFIG_SHELL_PATH}/_functions"); do
    source "$file"
done

clhist() {
    history -w && history -c && echo "" >| "$HISTFILE"
}

mkcd() {
    mkdir -p "$1" && cd "$1"
}
################################################

################################################
## KEYBINDS
################################################
for file in $(fd -e bash -t f . "${UNISHELL_CONFIG_SHELL_PATH}/_keybinds"); do
    source "$file"
done
################################################



################################################
## SHELL OPTS
################################################
### TERMINAL COLORING | Reference For Below: https://wiki.archlinux.org/title/readline
bind 'set colored-stats On'                 # Color files by types | NOTE: that this may cause completion text blink in some terminals (e.g. xterm).
bind 'set visible-stats On'                 # Append char to indicate type
bind 'set mark-symlinked-directories On'    # Mark symlinked directories+
bind 'set colored-completion-prefix On'     # Color the common prefix
bind 'set menu-complete-display-prefix On'  # Color the common prefix in menu-complete

### HISTORY
shopt -s histappend                         # append to the history file on shell exit instead of overwriting it.
shopt -s histverify                         # Allow editing of history-expanded lines before execution
shopt -s cmdhist                            # Save multi-line commands as a single entry in history
shopt -s lithist                            # Save multi-line command lines with line breaks in history
HISTSIZE=10000
HISTFILESIZE=20000
HISTCONTROL=ignoreboth
export HISTIGNORE="nuke:exit"
export HISTFILE="/var/tmp/.lesshist.swp"
export PROMPT_COMMAND="history -a; $PROMPT_COMMAND"    # history is shared near-live across sessions rather than only on close

### CD
shopt -s autocd                             # Allow changing directories by just typing the name (no need for `cd`)
# shopt -s cdspell                          # Automatically correct directory name typos when using `cd`
# shopt -s cdable_vars                      # Allow `cd` into variables that contain directory paths (e.g., `cd mydirvar`)

### EXPANSION
shopt -s extglob                            # Enable extended pattern matching operators (e.g., @(foo|bar))
shopt -s dotglob                            # Include hidden files (dotfiles) in pathname expansions
shopt -s nocaseglob                         # Make pathname expansion case-insensitive (e.g., matches *.TXT and *.txt)

### MISC
set -C                                      # DON'T OVERWRITE FILES WHEN REDIRECTING OUTPUTS (cmd > file) | WHEN SURE ABOUT OVERWRITING USE ">|" instead of ">"
bind Space:magic-space                      # AUTO EXPAND "!!"
shopt -s checkwinsize                       # Automatically update LINES and COLUMNS after each command (for proper window resizing)
shopt -s globstar                           # Enable recursive globbing with ** (e.g., **/*.txt)
shopt -s dirspell                           # Correct minor spelling errors in directory names during completion
shopt -s checkjobs                          # Warn if there are running background jobs when exiting shell
shopt -s checkhash                          # Re-check hashed command paths if command is not found
# shopt -s direxpand                        # Expand directory names in command line input (e.g., typing dir<TAB> shows full path)

################################################



################################################
## PROGRAMS-RELATED CONFIG
################################################

### VIM
vimrc="$UNISHELL_CONFIG_PATH/vim/vimrc"

if [[ -f $vimrc ]]; then
    for a in v vi vim; do
        alias "$a"="vim -c 'source $vimrc' "
    done
    alias sv="sudo --preserve-env=$SUDO_PRESERVED_VARIABLES vim -c 'source $vimrc' "
else
    echo -e "\nFailed to set VIM aliases. VIMRC file not found\n" >&2
fi

################################################



# vim: set ft=bash:
