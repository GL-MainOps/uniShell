# --- Elevate safely ----------------------------------------------------------
function es
    sudo --preserve-env=$SUDO_PRESERVED_VARIABLES \
        fish --init-command "source '$UNISHELL_CONFIG_PATH/shell-generated/$UNISHELL_SESSION_SHELL_PROFILE.fish'"
end

# --- DOUBLE ESC = toggle sudo on the current or previous command--------------
function _sudo_add_sudo
    set -l line (commandline)

    if test -z "$line"
        set line (string trim -- (history --max=1))
        commandline -- "$line"
        commandline -f end-of-line
    end

    if string match -q 'sudo *' -- "$line"
        set line (string replace -r '^sudo ' '' -- "$line")
    else
        set line "sudo $line"
    end

    commandline -- "$line"
    commandline -f end-of-line
end

bind escape,escape _sudo_add_sudo
bind --mode insert escape,escape _sudo_add_sudo
bind --mode default escape,escape _sudo_add_sudo
# vim: set ft=fish:
