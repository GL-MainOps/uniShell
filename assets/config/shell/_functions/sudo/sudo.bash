
# --- Elevate safely ----------------------------------------------------------
es() {
    sudo --preserve-env=$SUDO_PRESERVED_VARIABLES bash --rcfile "$UNISHELL_CONFIG_PATH/shell-generated/$UNISHELL_SESSION_SHELL_PROFILE.bash"
}

# --- DOUBLE ESC = SUDO <previous command> -----------------------------------
function _sudo_add_sudo {
    if [[ -z $READLINE_LINE ]]; then
        READLINE_LINE=$(fc -ln -1)
        READLINE_LINE="${READLINE_LINE#"${READLINE_LINE%%[![:space:]]*}"}"
        READLINE_POINT=${#READLINE_LINE}
        [[ $READLINE_LINE == 'sudo '* ]] && return
    fi

    if [[ $READLINE_LINE == 'sudo '* ]]; then
        READLINE_LINE=${READLINE_LINE#sudo }
        ((READLINE_POINT -= 5))
    else
        READLINE_LINE="sudo $READLINE_LINE"
        ((READLINE_POINT += 5))
    fi

    ((READLINE_POINT < 0)) && READLINE_POINT=0
}

bind -m emacs -x '"\e\e": _sudo_add_sudo'
bind -m vi-insert -x '"\e\e": _sudo_add_sudo'
bind -m vi-command -x '"\e\e": _sudo_add_sudo'
# vim: set ft=bash:
