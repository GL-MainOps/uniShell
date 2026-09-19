# ============================================================
# Ultra-light Bash prompt
#
# Features:
#   - Root-aware left prompt
#   - Root warning in right prompt
#   - Right-aligned Git branch
#   - Nerd Font Git icon
#   - Nerd Font distro icon
#   - Distro-specific brand color
#   - Current time
#   - Configurable colors
#   - Configurable text attributes
#   - Git repository discovery cached by PWD
#   - No external commands in the prompt hot path
#   - Readline-safe history navigation
#   - Readline-safe cursor movement/editing
#
# ============================================================
#
# ARCHITECTURE
# ============================================================
#
# The prompt has two parts:
#
#     LEFT PROMPT
#     RIGHT PROMPT
#
# The left prompt is constructed once.
#
# The right prompt is dynamic because:
#
#     - Git branch can change
#     - Time changes
#
# PROMPT_COMMAND therefore performs only the dynamic work:
#
#     1. Update Git branch
#     2. Generate right-prompt text
#     3. Update PS1
#
# The right prompt is NOT printed directly with printf.
#
# Instead, it becomes part of PS1 so Bash/Readline knows about
# it when redrawing the command line.
#
# ============================================================
#
# PERFORMANCE
# ============================================================
#
# No external commands are used in the normal prompt hot path:
#
#     git
#     date
#     find
#     fd
#     sed
#     awk
#     tput
#
# Git repository discovery happens only when PWD changes.
#
# Git HEAD is read directly with Bash's builtin `read`.
#
# Time is generated with Bash's builtin printf.
#
# Distro detection happens once when this file is sourced.
#
# ============================================================


shopt -s checkwinsize


# ============================================================
# CONFIG
# ============================================================


# ------------------------------------------------------------
# Segment visibility
# ------------------------------------------------------------
#
# 1 = enabled
# 0 = disabled
#
PROMPT_SHOW_ROOT=1
PROMPT_SHOW_GIT=1
PROMPT_SHOW_DISTRO=1
PROMPT_SHOW_TIME=1


# ------------------------------------------------------------
# Root warning
# ------------------------------------------------------------

PROMPT_ROOT_TEXT='⚠️ ROOT MODE ⚠️'


# ------------------------------------------------------------
# Time format
# ------------------------------------------------------------
#
# Examples:
#
#   '%a %d, %I:%M %p'
#       Thu 17, 02:01 PM
#
#   '%Y-%m-%d %H:%M:%S'
#       2026-09-17 14:01:23
#
#   '%H:%M'
#       14:01
#
#   '%I:%M %p'
#       02:01 PM
#
PROMPT_TIME_FORMAT='%a %d, %I:%M %p'


# ------------------------------------------------------------
# Separator
# ------------------------------------------------------------
#
# Example:
#
#     Git  |  Distro  |  Time
#
PROMPT_SEPARATOR='  |  '


# ============================================================
# LEFT PROMPT COLORS
# ============================================================


# ------------------------------------------------------------
# Brackets
# ------------------------------------------------------------

PROMPT_FG_BRACKETS='\033[01;37m'


# ------------------------------------------------------------
# Username
# ------------------------------------------------------------
#
# Normal user:
#
#     bright green
#
# Root:
#
#     bright red
#
PROMPT_FG_USER='\033[01;32m'
PROMPT_FG_ROOT='\033[01;31m'


# ------------------------------------------------------------
# @ separator
# ------------------------------------------------------------

PROMPT_FG_AT='\033[37m'


# ------------------------------------------------------------
# Hostname
# ------------------------------------------------------------

PROMPT_FG_HOST='\033[38;5;214m'


# ------------------------------------------------------------
# Current directory
# ------------------------------------------------------------

PROMPT_FG_PATH='\033[01;34m'


# ------------------------------------------------------------
# Prompt symbol
# ------------------------------------------------------------

PROMPT_FG_SYMBOL='\033[01;37m'
PROMPT_BG_SYMBOL='\033[48;5;52m'


# ============================================================
# RIGHT PROMPT TEXT FORMATTING
# ============================================================
#
# Formatting and colors are deliberately separate.
#
# This allows you to change:
#
#     bold
#     italic
#     underline
#     dim
#
# independently from colors.
#
# ============================================================


# ------------------------------------------------------------
# Right-prompt text attributes
# ------------------------------------------------------------
#
# Current:
#
#     BOLD
#
# ANSI:
#
#     1 = bold
#
# Examples:
#
#     Normal:
#         PROMPT_ATTR_RIGHT=''
#
#     Bold:
#         PROMPT_ATTR_RIGHT='\033[1m'
#
#     Dim:
#         PROMPT_ATTR_RIGHT='\033[2m'
#
#     Italic:
#         PROMPT_ATTR_RIGHT='\033[3m'
#
#     Underline:
#         PROMPT_ATTR_RIGHT='\033[4m'
#
#     Bold + italic:
#         PROMPT_ATTR_RIGHT='\033[1;3m'
#
#     Bold + underline:
#         PROMPT_ATTR_RIGHT='\033[1;4m'
#
PROMPT_ATTR_RIGHT='\033[1m'


# ------------------------------------------------------------
# Right-prompt attribute reset
# ------------------------------------------------------------
#
# 22 = normal intensity.
#
PROMPT_ATTR_RESET='\033[22m'


# ============================================================
# RIGHT PROMPT COLORS
# ============================================================


# ------------------------------------------------------------
# Root warning
# ------------------------------------------------------------

PROMPT_FG_ROOT_MODE='\033[01;31m'


# ------------------------------------------------------------
# Git
# ------------------------------------------------------------

PROMPT_FG_GIT='\033[01;32m'


# ------------------------------------------------------------
# Distro
# ------------------------------------------------------------

PROMPT_FG_DISTRO='\033[01;37m'
PROMPT_FG_DISTRO_BRAND='\033[01;37m'


# ------------------------------------------------------------
# Time
# ------------------------------------------------------------

PROMPT_FG_TIME='\033[01;37m'


# ------------------------------------------------------------
# Separators
# ------------------------------------------------------------

PROMPT_FG_SEPARATOR='\033[01;37m'


# ------------------------------------------------------------
# Full ANSI reset
# ------------------------------------------------------------

PROMPT_RESET='\033[00m'


# ============================================================
# NERD FONT ICONS
# ============================================================

PROMPT_GIT_ICON='󰊢'
PROMPT_DISTRO_ICON='󰣖'


# ============================================================
# ROOT MODE
# ============================================================
#
# Root status is static for the lifetime of the shell.
#
# IMPORTANT:
#
# Do NOT store a boolean such as:
#
#     PROMPT_ROOT_MODE=1
#
# and then accidentally use that as display content.
#
# Instead, cache the actual formatted root-warning string only
# when this shell is really running as root.
# ============================================================

PROMPT_ROOT_MODE=''

if (( EUID == 0 && PROMPT_SHOW_ROOT )); then
    PROMPT_ROOT_MODE="${PROMPT_ATTR_RIGHT}${PROMPT_FG_ROOT_MODE}${PROMPT_ROOT_TEXT}${PROMPT_RESET}${PROMPT_ATTR_RESET}"
fi


# ============================================================
# LEFT PROMPT
# ============================================================
#
# The left prompt is static and therefore built only once.
#
# Bash expands:
#
#     \u  -> username
#     \h  -> hostname
#     \w  -> current directory
#     \$  -> '$' for normal users, '#' for root
#
# There is deliberately a literal SPACE immediately after:
#
#     \$
#
# Therefore:
#
#     normal user:
#         $ _
#
#     root:
#         # _
#
# where `_` represents the command-entry cursor.
# ============================================================

_prompt_build_left() {
    local user_color

    if (( EUID == 0 )); then
        user_color=$PROMPT_FG_ROOT
    else
        user_color=$PROMPT_FG_USER
    fi

    printf -v PROMPT_LEFT \
        '\[%s\][\[%s\]\\u\[%s\]@\[%s\]\\h \[%s\]\\w\[%s\]] \[%s\]\[%s\] \\$ \[%s\] ' \
        "$PROMPT_FG_BRACKETS" \
        "$user_color" \
        "$PROMPT_FG_AT" \
        "$PROMPT_FG_HOST" \
        "$PROMPT_FG_PATH" \
        "$PROMPT_FG_BRACKETS" \
        "$PROMPT_BG_SYMBOL" \
        "$PROMPT_FG_SYMBOL" \
        "$PROMPT_RESET"
}

_prompt_build_left


# ============================================================
# DISTRO DETECTION
# ============================================================
#
# Runs once when this configuration is sourced.
# ============================================================

_prompt_detect_distro() {
    [[ $PROMPT_SHOW_DISTRO -eq 1 ]] || return
    [[ -r /etc/os-release ]] || return

    local id
    local id_like

    # shellcheck disable=SC1091
    . /etc/os-release

    id=${ID,,}
    id_like=${ID_LIKE,,}

    case "$id" in

        ubuntu)
            PROMPT_DISTRO_ICON='󰕈'
            PROMPT_FG_DISTRO_BRAND='\033[01;38;5;208m'
            ;;

        debian)
            PROMPT_DISTRO_ICON='󰣚'
            PROMPT_FG_DISTRO_BRAND='\033[01;38;5;160m'
            ;;

        fedora)
            PROMPT_DISTRO_ICON='󰣛'
            PROMPT_FG_DISTRO_BRAND='\033[01;38;5;39m'
            ;;

        arch)
            PROMPT_DISTRO_ICON='󰣇'
            PROMPT_FG_DISTRO_BRAND='\033[01;38;5;75m'
            ;;

        rhel)
            PROMPT_DISTRO_ICON='󱄛'
            PROMPT_FG_DISTRO_BRAND='\033[01;38;5;196m'
            ;;

        *)
            case "$id_like" in

                *ubuntu*)
                    PROMPT_DISTRO_ICON='󰕈'
                    PROMPT_FG_DISTRO_BRAND='\033[01;38;5;208m'
                    ;;

                *debian*)
                    PROMPT_DISTRO_ICON='󰣚'
                    PROMPT_FG_DISTRO_BRAND='\033[01;38;5;160m'
                    ;;

                *fedora*)
                    PROMPT_DISTRO_ICON='󰣛'
                    PROMPT_FG_DISTRO_BRAND='\033[01;38;5;39m'
                    ;;

                *arch*)
                    PROMPT_DISTRO_ICON='󰣇'
                    PROMPT_FG_DISTRO_BRAND='\033[01;38;5;75m'
                    ;;

                *rhel*|*redhat*)
                    PROMPT_DISTRO_ICON='󱄛'
                    PROMPT_FG_DISTRO_BRAND='\033[01;38;5;196m'
                    ;;

            esac
            ;;
    esac
}

_prompt_detect_distro


# ============================================================
# GIT CACHE
# ============================================================

_PROMPT_GIT_CACHE_PWD=''
_PROMPT_GIT_CACHE_DIR=''
PROMPT_GIT_BRANCH=''


# ------------------------------------------------------------
# Locate Git repository metadata
# ------------------------------------------------------------

_prompt_find_git_dir() {
    local dir=$PWD
    local git_entry
    local line
    local gitdir

    while :; do

        git_entry="$dir/.git"

        # Normal Git repository.
        if [[ -d $git_entry ]]; then
            _PROMPT_GIT_CACHE_DIR=$git_entry
            return
        fi

        # Git worktree / linked repository.
        if [[ -f $git_entry ]]; then

            IFS= read -r line < "$git_entry"

            if [[ $line == 'gitdir: '* ]]; then

                gitdir=${line#gitdir: }

                [[ $gitdir == /* ]] || \
                    gitdir="$dir/$gitdir"

                _PROMPT_GIT_CACHE_DIR=$gitdir
                return
            fi
        fi

        [[ $dir == / ]] && return

        dir=${dir%/*}

        [[ -z $dir ]] && dir=/
    done
}


# ------------------------------------------------------------
# Update Git branch
# ------------------------------------------------------------

_prompt_update_git() {
    local head

    PROMPT_GIT_BRANCH=''

    # Only search for the repository when PWD changes.
    if [[ $_PROMPT_GIT_CACHE_PWD != "$PWD" ]]; then

        _PROMPT_GIT_CACHE_PWD=$PWD
        _PROMPT_GIT_CACHE_DIR=''

        _prompt_find_git_dir
    fi

    [[ -n $_PROMPT_GIT_CACHE_DIR ]] || return
    [[ -r $_PROMPT_GIT_CACHE_DIR/HEAD ]] || return

    # Read HEAD directly instead of invoking `git`.
    IFS= read -r head < "$_PROMPT_GIT_CACHE_DIR/HEAD"

    if [[ $head == 'ref: refs/heads/'* ]]; then

        PROMPT_GIT_BRANCH=${head#ref: refs/heads/}

    elif [[ $head =~ ^[0-9a-fA-F]{40}$ ]]; then

        # Detached HEAD.
        PROMPT_GIT_BRANCH=${head:0:7}

    fi
}


# ============================================================
# RIGHT PROMPT BUILDER
# ============================================================
#
# This function does NOT print anything.
#
# It creates:
#
#     PROMPT_RIGHT
#
# which is incorporated into PS1.
# ============================================================

_prompt_build_right() {
    local time_text=''
    local colored=''
    local plain=''

    # --------------------------------------------------------
    # ROOT MODE
    # --------------------------------------------------------

    if [[ -n $PROMPT_ROOT_MODE ]]; then

        colored+="$PROMPT_ROOT_MODE"
        plain+="$PROMPT_ROOT_TEXT"

    fi


    # --------------------------------------------------------
    # GIT
    # --------------------------------------------------------

    if (( PROMPT_SHOW_GIT )); then

        _prompt_update_git

        if [[ -n $PROMPT_GIT_BRANCH ]]; then

            if [[ -n $plain ]]; then

                colored+="${PROMPT_ATTR_RIGHT}"
                colored+="${PROMPT_FG_SEPARATOR}"
                colored+="${PROMPT_SEPARATOR}"
                colored+="${PROMPT_RESET}"
                colored+="${PROMPT_ATTR_RESET}"

                plain+="${PROMPT_SEPARATOR}"
            fi

            colored+="${PROMPT_ATTR_RIGHT}"
            colored+="${PROMPT_FG_GIT}"
            colored+="${PROMPT_GIT_ICON} ${PROMPT_GIT_BRANCH}"
            colored+="${PROMPT_RESET}"
            colored+="${PROMPT_ATTR_RESET}"

            plain+="${PROMPT_GIT_ICON} ${PROMPT_GIT_BRANCH}"
        fi
    fi


    # --------------------------------------------------------
    # DISTRO
    # --------------------------------------------------------

    if (( PROMPT_SHOW_DISTRO )); then

        if [[ -n $plain ]]; then

            colored+="${PROMPT_ATTR_RIGHT}"
            colored+="${PROMPT_FG_SEPARATOR}"
            colored+="${PROMPT_SEPARATOR}"
            colored+="${PROMPT_RESET}"
            colored+="${PROMPT_ATTR_RESET}"

            plain+="${PROMPT_SEPARATOR}"
        fi

        colored+="${PROMPT_ATTR_RIGHT}"
        colored+="${PROMPT_FG_DISTRO_BRAND}"
        colored+="${PROMPT_DISTRO_ICON}"
        colored+="${PROMPT_RESET}"
        colored+="${PROMPT_ATTR_RESET}"

        plain+="${PROMPT_DISTRO_ICON}"
    fi


    # --------------------------------------------------------
    # TIME
    # --------------------------------------------------------

    if (( PROMPT_SHOW_TIME )); then

        printf -v time_text "%(${PROMPT_TIME_FORMAT})T" -1

        if [[ -n $plain ]]; then

            colored+="${PROMPT_ATTR_RIGHT}"
            colored+="${PROMPT_FG_SEPARATOR}"
            colored+="${PROMPT_SEPARATOR}"
            colored+="${PROMPT_RESET}"
            colored+="${PROMPT_ATTR_RESET}"

            plain+="${PROMPT_SEPARATOR}"
        fi

        colored+="${PROMPT_ATTR_RIGHT}"
        colored+="${PROMPT_FG_TIME}"
        colored+="${time_text}"
        colored+="${PROMPT_RESET}"
        colored+="${PROMPT_ATTR_RESET}"

        plain+="${time_text}"
    fi


    # --------------------------------------------------------
    # Nothing to render
    # --------------------------------------------------------

    if [[ -z $plain ]]; then

        PROMPT_RIGHT=''
        return

    fi


    # --------------------------------------------------------
    # Calculate visible width
    # --------------------------------------------------------

    local width=${#plain}
    local start_column

    (( COLUMNS > 0 )) || {
        PROMPT_RIGHT=''
        return
    }

    (( width < COLUMNS )) || {
        PROMPT_RIGHT=''
        return
    }

    start_column=$((COLUMNS - width + 1))


    # --------------------------------------------------------
    # Readline-safe cursor positioning
    # --------------------------------------------------------
    #
    # ASCII 0x01 and 0x02 tell Readline that the enclosed
    # terminal control sequences occupy zero screen columns.
    #
    # This is critical for correct:
    #
    #     Up
    #     Down
    #     Home
    #     End
    #     Left
    #     Right
    #
    # behavior.
    # --------------------------------------------------------

    printf -v PROMPT_RIGHT \
        '\001\033[s\033[%dG\002%b\001\033[u\002' \
        "$start_column" \
        "$colored"
}


# ============================================================
# PROMPT UPDATE
# ============================================================
#
# PROMPT_COMMAND calls this before Bash displays PS1.
#
# It calculates the dynamic right prompt and then places it
# inside PS1.
# ============================================================

_prompt_update() {
    _prompt_build_right

    PS1="${PROMPT_RIGHT}${PROMPT_LEFT}"
}


# ============================================================
# PROMPT_COMMAND REGISTRATION
# ============================================================
#
# Supports both:
#
#     PROMPT_COMMAND='command'
#
# and:
#
#     PROMPT_COMMAND=(command1 command2 ...)
#
# Avoids registering the function twice when this file is
# sourced repeatedly.
# ============================================================

if declare -p PROMPT_COMMAND 2>/dev/null | grep -q '^declare -a'; then

    [[ " ${PROMPT_COMMAND[*]} " == *" _prompt_update "* ]] ||
        PROMPT_COMMAND+=(_prompt_update)

else

    [[ $PROMPT_COMMAND == *_prompt_update* ]] ||
        PROMPT_COMMAND="_prompt_update${PROMPT_COMMAND:+;$PROMPT_COMMAND}"

fi

