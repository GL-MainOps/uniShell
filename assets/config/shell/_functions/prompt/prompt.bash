
if (( EUID == 0 )); then
    export PS1="$ROOT_PROMPT"
else
    export PS1="$USER_PROMPT"
fi

#X# # ============================================================
#X# # Ultra-light Bash prompt
#X# #
#X# # Features:
#X# #   - Root-aware left prompt
#X# #   - Root warning in right prompt
#X# #   - Right-aligned Git branch
#X# #   - Nerd Font Git icon
#X# #   - Nerd Font distro icon
#X# #   - Distro-specific brand color
#X# #   - Current time
#X# #   - Configurable colors
#X# #   - Configurable text attributes
#X# #   - Git repository discovery cached by PWD
#X# #   - No external commands in the prompt hot path
#X# #   - Readline-safe history navigation
#X# #   - Readline-safe cursor movement/editing
#X# #
#X# # ============================================================
#X# #
#X# # ARCHITECTURE
#X# # ============================================================
#X# #
#X# # The prompt has two parts:
#X# #
#X# #     LEFT PROMPT
#X# #     RIGHT PROMPT
#X# #
#X# # The left prompt is constructed once.
#X# #
#X# # The right prompt is dynamic because:
#X# #
#X# #     - Git branch can change
#X# #     - Time changes
#X# #
#X# # PROMPT_COMMAND therefore performs only the dynamic work:
#X# #
#X# #     1. Update Git branch
#X# #     2. Generate right-prompt text
#X# #     3. Update PS1
#X# #
#X# # The right prompt is NOT printed directly with printf.
#X# #
#X# # Instead, it becomes part of PS1 so Bash/Readline knows about
#X# # it when redrawing the command line.
#X# #
#X# # ============================================================
#X# #
#X# # PERFORMANCE
#X# # ============================================================
#X# #
#X# # No external commands are used in the normal prompt hot path:
#X# #
#X# #     git
#X# #     date
#X# #     find
#X# #     fd
#X# #     sed
#X# #     awk
#X# #     tput
#X# #
#X# # Git repository discovery happens only when PWD changes.
#X# #
#X# # Git HEAD is read directly with Bash's builtin `read`.
#X# #
#X# # Time is generated with Bash's builtin printf.
#X# #
#X# # Distro detection happens once when this file is sourced.
#X# #
#X# # ============================================================
#X# 
#X# 
#X# shopt -s checkwinsize
#X# 
#X# 
#X# # ============================================================
#X# # CONFIG
#X# # ============================================================
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Segment visibility
#X# # ------------------------------------------------------------
#X# #
#X# # 1 = enabled
#X# # 0 = disabled
#X# #
#X# PROMPT_SHOW_ROOT=1
#X# PROMPT_SHOW_GIT=1
#X# PROMPT_SHOW_DISTRO=1
#X# PROMPT_SHOW_TIME=1
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Root warning
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_ROOT_TEXT='⚠️ ROOT MODE ⚠️'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Time format
#X# # ------------------------------------------------------------
#X# #
#X# # Examples:
#X# #
#X# #   '%a %d, %I:%M %p'
#X# #       Thu 17, 02:01 PM
#X# #
#X# #   '%Y-%m-%d %H:%M:%S'
#X# #       2026-09-17 14:01:23
#X# #
#X# #   '%H:%M'
#X# #       14:01
#X# #
#X# #   '%I:%M %p'
#X# #       02:01 PM
#X# #
#X# PROMPT_TIME_FORMAT='%a %d, %I:%M %p'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Separator
#X# # ------------------------------------------------------------
#X# #
#X# # Example:
#X# #
#X# #     Git  |  Distro  |  Time
#X# #
#X# PROMPT_SEPARATOR='  |  '
#X# 
#X# 
#X# # ============================================================
#X# # LEFT PROMPT COLORS
#X# # ============================================================
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Brackets
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_BRACKETS='\033[01;37m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Username
#X# # ------------------------------------------------------------
#X# #
#X# # Normal user:
#X# #
#X# #     bright green
#X# #
#X# # Root:
#X# #
#X# #     bright red
#X# #
#X# PROMPT_FG_USER='\033[01;32m'
#X# PROMPT_FG_ROOT='\033[01;31m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # @ separator
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_AT='\033[37m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Hostname
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_HOST='\033[38;5;214m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Current directory
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_PATH='\033[01;34m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Prompt symbol
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_SYMBOL='\033[01;37m'
#X# PROMPT_BG_SYMBOL='\033[48;5;52m'
#X# 
#X# 
#X# # ============================================================
#X# # RIGHT PROMPT TEXT FORMATTING
#X# # ============================================================
#X# #
#X# # Formatting and colors are deliberately separate.
#X# #
#X# # This allows you to change:
#X# #
#X# #     bold
#X# #     italic
#X# #     underline
#X# #     dim
#X# #
#X# # independently from colors.
#X# #
#X# # ============================================================
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Right-prompt text attributes
#X# # ------------------------------------------------------------
#X# #
#X# # Current:
#X# #
#X# #     BOLD
#X# #
#X# # ANSI:
#X# #
#X# #     1 = bold
#X# #
#X# # Examples:
#X# #
#X# #     Normal:
#X# #         PROMPT_ATTR_RIGHT=''
#X# #
#X# #     Bold:
#X# #         PROMPT_ATTR_RIGHT='\033[1m'
#X# #
#X# #     Dim:
#X# #         PROMPT_ATTR_RIGHT='\033[2m'
#X# #
#X# #     Italic:
#X# #         PROMPT_ATTR_RIGHT='\033[3m'
#X# #
#X# #     Underline:
#X# #         PROMPT_ATTR_RIGHT='\033[4m'
#X# #
#X# #     Bold + italic:
#X# #         PROMPT_ATTR_RIGHT='\033[1;3m'
#X# #
#X# #     Bold + underline:
#X# #         PROMPT_ATTR_RIGHT='\033[1;4m'
#X# #
#X# PROMPT_ATTR_RIGHT='\033[1m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Right-prompt attribute reset
#X# # ------------------------------------------------------------
#X# #
#X# # 22 = normal intensity.
#X# #
#X# PROMPT_ATTR_RESET='\033[22m'
#X# 
#X# 
#X# # ============================================================
#X# # RIGHT PROMPT COLORS
#X# # ============================================================
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Root warning
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_ROOT_MODE='\033[01;31m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Git
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_GIT='\033[01;32m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Distro
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_DISTRO='\033[01;37m'
#X# PROMPT_FG_DISTRO_BRAND='\033[01;37m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Time
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_TIME='\033[01;37m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Separators
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_FG_SEPARATOR='\033[01;37m'
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Full ANSI reset
#X# # ------------------------------------------------------------
#X# 
#X# PROMPT_RESET='\033[00m'
#X# 
#X# 
#X# # ============================================================
#X# # NERD FONT ICONS
#X# # ============================================================
#X# 
#X# PROMPT_GIT_ICON='󰊢'
#X# PROMPT_DISTRO_ICON='󰣖'
#X# 
#X# 
#X# # ============================================================
#X# # ROOT MODE
#X# # ============================================================
#X# #
#X# # Root status is static for the lifetime of the shell.
#X# #
#X# # IMPORTANT:
#X# #
#X# # Do NOT store a boolean such as:
#X# #
#X# #     PROMPT_ROOT_MODE=1
#X# #
#X# # and then accidentally use that as display content.
#X# #
#X# # Instead, cache the actual formatted root-warning string only
#X# # when this shell is really running as root.
#X# # ============================================================
#X# 
#X# PROMPT_ROOT_MODE=''
#X# 
#X# if (( EUID == 0 && PROMPT_SHOW_ROOT )); then
#X#     PROMPT_ROOT_MODE="${PROMPT_ATTR_RIGHT}${PROMPT_FG_ROOT_MODE}${PROMPT_ROOT_TEXT}${PROMPT_RESET}${PROMPT_ATTR_RESET}"
#X# fi
#X# 
#X# 
#X# # ============================================================
#X# # LEFT PROMPT
#X# # ============================================================
#X# #
#X# # The left prompt is static and therefore built only once.
#X# #
#X# # Bash expands:
#X# #
#X# #     \u  -> username
#X# #     \h  -> hostname
#X# #     \w  -> current directory
#X# #     \$  -> '$' for normal users, '#' for root
#X# #
#X# # There is deliberately a literal SPACE immediately after:
#X# #
#X# #     \$
#X# #
#X# # Therefore:
#X# #
#X# #     normal user:
#X# #         $ _
#X# #
#X# #     root:
#X# #         # _
#X# #
#X# # where `_` represents the command-entry cursor.
#X# # ============================================================
#X# 
#X# _prompt_build_left() {
#X#     local user_color
#X# 
#X#     if (( EUID == 0 )); then
#X#         user_color=$PROMPT_FG_ROOT
#X#     else
#X#         user_color=$PROMPT_FG_USER
#X#     fi
#X# 
#X#     printf -v PROMPT_LEFT \
#X#         '\[%s\][\[%s\]\\u\[%s\]@\[%s\]\\h \[%s\]\\w\[%s\]] \[%s\]\[%s\] \\$ \[%s\] ' \
#X#         "$PROMPT_FG_BRACKETS" \
#X#         "$user_color" \
#X#         "$PROMPT_FG_AT" \
#X#         "$PROMPT_FG_HOST" \
#X#         "$PROMPT_FG_PATH" \
#X#         "$PROMPT_FG_BRACKETS" \
#X#         "$PROMPT_BG_SYMBOL" \
#X#         "$PROMPT_FG_SYMBOL" \
#X#         "$PROMPT_RESET"
#X# }
#X# 
#X# _prompt_build_left
#X# 
#X# 
#X# # ============================================================
#X# # DISTRO DETECTION
#X# # ============================================================
#X# #
#X# # Runs once when this configuration is sourced.
#X# # ============================================================
#X# 
#X# _prompt_detect_distro() {
#X#     [[ $PROMPT_SHOW_DISTRO -eq 1 ]] || return
#X#     [[ -r /etc/os-release ]] || return
#X# 
#X#     local id
#X#     local id_like
#X# 
#X#     # shellcheck disable=SC1091
#X#     . /etc/os-release
#X# 
#X#     id=${ID,,}
#X#     id_like=${ID_LIKE,,}
#X# 
#X#     case "$id" in
#X# 
#X#         ubuntu)
#X#             PROMPT_DISTRO_ICON='󰕈'
#X#             PROMPT_FG_DISTRO_BRAND='\033[01;38;5;208m'
#X#             ;;
#X# 
#X#         debian)
#X#             PROMPT_DISTRO_ICON='󰣚'
#X#             PROMPT_FG_DISTRO_BRAND='\033[01;38;5;160m'
#X#             ;;
#X# 
#X#         fedora)
#X#             PROMPT_DISTRO_ICON='󰣛'
#X#             PROMPT_FG_DISTRO_BRAND='\033[01;38;5;39m'
#X#             ;;
#X# 
#X#         arch)
#X#             PROMPT_DISTRO_ICON='󰣇'
#X#             PROMPT_FG_DISTRO_BRAND='\033[01;38;5;75m'
#X#             ;;
#X# 
#X#         rhel)
#X#             PROMPT_DISTRO_ICON='󱄛'
#X#             PROMPT_FG_DISTRO_BRAND='\033[01;38;5;196m'
#X#             ;;
#X# 
#X#         *)
#X#             case "$id_like" in
#X# 
#X#                 *ubuntu*)
#X#                     PROMPT_DISTRO_ICON='󰕈'
#X#                     PROMPT_FG_DISTRO_BRAND='\033[01;38;5;208m'
#X#                     ;;
#X# 
#X#                 *debian*)
#X#                     PROMPT_DISTRO_ICON='󰣚'
#X#                     PROMPT_FG_DISTRO_BRAND='\033[01;38;5;160m'
#X#                     ;;
#X# 
#X#                 *fedora*)
#X#                     PROMPT_DISTRO_ICON='󰣛'
#X#                     PROMPT_FG_DISTRO_BRAND='\033[01;38;5;39m'
#X#                     ;;
#X# 
#X#                 *arch*)
#X#                     PROMPT_DISTRO_ICON='󰣇'
#X#                     PROMPT_FG_DISTRO_BRAND='\033[01;38;5;75m'
#X#                     ;;
#X# 
#X#                 *rhel*|*redhat*)
#X#                     PROMPT_DISTRO_ICON='󱄛'
#X#                     PROMPT_FG_DISTRO_BRAND='\033[01;38;5;196m'
#X#                     ;;
#X# 
#X#             esac
#X#             ;;
#X#     esac
#X# }
#X# 
#X# _prompt_detect_distro
#X# 
#X# 
#X# # ============================================================
#X# # GIT CACHE
#X# # ============================================================
#X# 
#X# _PROMPT_GIT_CACHE_PWD=''
#X# _PROMPT_GIT_CACHE_DIR=''
#X# PROMPT_GIT_BRANCH=''
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Locate Git repository metadata
#X# # ------------------------------------------------------------
#X# 
#X# _prompt_find_git_dir() {
#X#     local dir=$PWD
#X#     local git_entry
#X#     local line
#X#     local gitdir
#X# 
#X#     while :; do
#X# 
#X#         git_entry="$dir/.git"
#X# 
#X#         # Normal Git repository.
#X#         if [[ -d $git_entry ]]; then
#X#             _PROMPT_GIT_CACHE_DIR=$git_entry
#X#             return
#X#         fi
#X# 
#X#         # Git worktree / linked repository.
#X#         if [[ -f $git_entry ]]; then
#X# 
#X#             IFS= read -r line < "$git_entry"
#X# 
#X#             if [[ $line == 'gitdir: '* ]]; then
#X# 
#X#                 gitdir=${line#gitdir: }
#X# 
#X#                 [[ $gitdir == /* ]] || \
#X#                     gitdir="$dir/$gitdir"
#X# 
#X#                 _PROMPT_GIT_CACHE_DIR=$gitdir
#X#                 return
#X#             fi
#X#         fi
#X# 
#X#         [[ $dir == / ]] && return
#X# 
#X#         dir=${dir%/*}
#X# 
#X#         [[ -z $dir ]] && dir=/
#X#     done
#X# }
#X# 
#X# 
#X# # ------------------------------------------------------------
#X# # Update Git branch
#X# # ------------------------------------------------------------
#X# 
#X# _prompt_update_git() {
#X#     local head
#X# 
#X#     PROMPT_GIT_BRANCH=''
#X# 
#X#     # Only search for the repository when PWD changes.
#X#     if [[ $_PROMPT_GIT_CACHE_PWD != "$PWD" ]]; then
#X# 
#X#         _PROMPT_GIT_CACHE_PWD=$PWD
#X#         _PROMPT_GIT_CACHE_DIR=''
#X# 
#X#         _prompt_find_git_dir
#X#     fi
#X# 
#X#     [[ -n $_PROMPT_GIT_CACHE_DIR ]] || return
#X#     [[ -r $_PROMPT_GIT_CACHE_DIR/HEAD ]] || return
#X# 
#X#     # Read HEAD directly instead of invoking `git`.
#X#     IFS= read -r head < "$_PROMPT_GIT_CACHE_DIR/HEAD"
#X# 
#X#     if [[ $head == 'ref: refs/heads/'* ]]; then
#X# 
#X#         PROMPT_GIT_BRANCH=${head#ref: refs/heads/}
#X# 
#X#     elif [[ $head =~ ^[0-9a-fA-F]{40}$ ]]; then
#X# 
#X#         # Detached HEAD.
#X#         PROMPT_GIT_BRANCH=${head:0:7}
#X# 
#X#     fi
#X# }
#X# 
#X# 
#X# # ============================================================
#X# # RIGHT PROMPT BUILDER
#X# # ============================================================
#X# #
#X# # This function does NOT print anything.
#X# #
#X# # It creates:
#X# #
#X# #     PROMPT_RIGHT
#X# #
#X# # which is incorporated into PS1.
#X# # ============================================================
#X# 
#X# _prompt_build_right() {
#X#     local time_text=''
#X#     local colored=''
#X#     local plain=''
#X# 
#X#     # --------------------------------------------------------
#X#     # ROOT MODE
#X#     # --------------------------------------------------------
#X# 
#X#     if [[ -n $PROMPT_ROOT_MODE ]]; then
#X# 
#X#         colored+="$PROMPT_ROOT_MODE"
#X#         plain+="$PROMPT_ROOT_TEXT"
#X# 
#X#     fi
#X# 
#X# 
#X#     # --------------------------------------------------------
#X#     # GIT
#X#     # --------------------------------------------------------
#X# 
#X#     if (( PROMPT_SHOW_GIT )); then
#X# 
#X#         _prompt_update_git
#X# 
#X#         if [[ -n $PROMPT_GIT_BRANCH ]]; then
#X# 
#X#             if [[ -n $plain ]]; then
#X# 
#X#                 colored+="${PROMPT_ATTR_RIGHT}"
#X#                 colored+="${PROMPT_FG_SEPARATOR}"
#X#                 colored+="${PROMPT_SEPARATOR}"
#X#                 colored+="${PROMPT_RESET}"
#X#                 colored+="${PROMPT_ATTR_RESET}"
#X# 
#X#                 plain+="${PROMPT_SEPARATOR}"
#X#             fi
#X# 
#X#             colored+="${PROMPT_ATTR_RIGHT}"
#X#             colored+="${PROMPT_FG_GIT}"
#X#             colored+="${PROMPT_GIT_ICON} ${PROMPT_GIT_BRANCH}"
#X#             colored+="${PROMPT_RESET}"
#X#             colored+="${PROMPT_ATTR_RESET}"
#X# 
#X#             plain+="${PROMPT_GIT_ICON} ${PROMPT_GIT_BRANCH}"
#X#         fi
#X#     fi
#X# 
#X# 
#X#     # --------------------------------------------------------
#X#     # DISTRO
#X#     # --------------------------------------------------------
#X# 
#X#     if (( PROMPT_SHOW_DISTRO )); then
#X# 
#X#         if [[ -n $plain ]]; then
#X# 
#X#             colored+="${PROMPT_ATTR_RIGHT}"
#X#             colored+="${PROMPT_FG_SEPARATOR}"
#X#             colored+="${PROMPT_SEPARATOR}"
#X#             colored+="${PROMPT_RESET}"
#X#             colored+="${PROMPT_ATTR_RESET}"
#X# 
#X#             plain+="${PROMPT_SEPARATOR}"
#X#         fi
#X# 
#X#         colored+="${PROMPT_ATTR_RIGHT}"
#X#         colored+="${PROMPT_FG_DISTRO_BRAND}"
#X#         colored+="${PROMPT_DISTRO_ICON}"
#X#         colored+="${PROMPT_RESET}"
#X#         colored+="${PROMPT_ATTR_RESET}"
#X# 
#X#         plain+="${PROMPT_DISTRO_ICON}"
#X#     fi
#X# 
#X# 
#X#     # --------------------------------------------------------
#X#     # TIME
#X#     # --------------------------------------------------------
#X# 
#X#     if (( PROMPT_SHOW_TIME )); then
#X# 
#X#         printf -v time_text "%(${PROMPT_TIME_FORMAT})T" -1
#X# 
#X#         if [[ -n $plain ]]; then
#X# 
#X#             colored+="${PROMPT_ATTR_RIGHT}"
#X#             colored+="${PROMPT_FG_SEPARATOR}"
#X#             colored+="${PROMPT_SEPARATOR}"
#X#             colored+="${PROMPT_RESET}"
#X#             colored+="${PROMPT_ATTR_RESET}"
#X# 
#X#             plain+="${PROMPT_SEPARATOR}"
#X#         fi
#X# 
#X#         colored+="${PROMPT_ATTR_RIGHT}"
#X#         colored+="${PROMPT_FG_TIME}"
#X#         colored+="${time_text}"
#X#         colored+="${PROMPT_RESET}"
#X#         colored+="${PROMPT_ATTR_RESET}"
#X# 
#X#         plain+="${time_text}"
#X#     fi
#X# 
#X# 
#X#     # --------------------------------------------------------
#X#     # Nothing to render
#X#     # --------------------------------------------------------
#X# 
#X#     if [[ -z $plain ]]; then
#X# 
#X#         PROMPT_RIGHT=''
#X#         return
#X# 
#X#     fi
#X# 
#X# 
#X#     # --------------------------------------------------------
#X#     # Calculate visible width
#X#     # --------------------------------------------------------
#X# 
#X#     local width=${#plain}
#X#     local start_column
#X# 
#X#     (( COLUMNS > 0 )) || {
#X#         PROMPT_RIGHT=''
#X#         return
#X#     }
#X# 
#X#     (( width < COLUMNS )) || {
#X#         PROMPT_RIGHT=''
#X#         return
#X#     }
#X# 
#X#     start_column=$((COLUMNS - width + 1))
#X# 
#X# 
#X#     # --------------------------------------------------------
#X#     # Readline-safe cursor positioning
#X#     # --------------------------------------------------------
#X#     #
#X#     # ASCII 0x01 and 0x02 tell Readline that the enclosed
#X#     # terminal control sequences occupy zero screen columns.
#X#     #
#X#     # This is critical for correct:
#X#     #
#X#     #     Up
#X#     #     Down
#X#     #     Home
#X#     #     End
#X#     #     Left
#X#     #     Right
#X#     #
#X#     # behavior.
#X#     # --------------------------------------------------------
#X# 
#X#     printf -v PROMPT_RIGHT \
#X#         '\001\033[s\033[%dG\002%b\001\033[u\002' \
#X#         "$start_column" \
#X#         "$colored"
#X# }
#X# 
#X# 
#X# # ============================================================
#X# # PROMPT UPDATE
#X# # ============================================================
#X# #
#X# # PROMPT_COMMAND calls this before Bash displays PS1.
#X# #
#X# # It calculates the dynamic right prompt and then places it
#X# # inside PS1.
#X# # ============================================================
#X# 
#X# _prompt_update() {
#X#     _prompt_build_right
#X# 
#X#     PS1="${PROMPT_RIGHT}${PROMPT_LEFT}"
#X# }
#X# 
#X# 
#X# # ============================================================
#X# # PROMPT_COMMAND REGISTRATION
#X# # ============================================================
#X# #
#X# # Supports both:
#X# #
#X# #     PROMPT_COMMAND='command'
#X# #
#X# # and:
#X# #
#X# #     PROMPT_COMMAND=(command1 command2 ...)
#X# #
#X# # Avoids registering the function twice when this file is
#X# # sourced repeatedly.
#X# # ============================================================
#X# 
#X# if declare -p PROMPT_COMMAND 2>/dev/null | grep -q '^declare -a'; then
#X# 
#X#     [[ " ${PROMPT_COMMAND[*]} " == *" _prompt_update "* ]] ||
#X#         PROMPT_COMMAND+=(_prompt_update)
#X# 
#X# else
#X# 
#X#     [[ $PROMPT_COMMAND == *_prompt_update* ]] ||
#X#         PROMPT_COMMAND="_prompt_update${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
#X# 
#X# fi
#X# 
