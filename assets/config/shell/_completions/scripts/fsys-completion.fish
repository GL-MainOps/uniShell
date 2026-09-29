# Fish completion for fsys / ffs
#
# Install:
#   mkdir -p ~/.config/fish/completions
#   cp fsys.fish.fixed ~/.config/fish/completions/fsys.fish
#
# The second command name (ffs) is intentional: it is the user's alias.

function __fsys_seen_command
    set -l tokens (commandline -opc)

    for token in $tokens[2..-1]
        switch $token
            case service services svc timer timers unit units unit-file unit-files \
                 failed log logs journal boot boots overview status linger
                return 0
        end
    end

    return 1
end

# Disable filesystem completion.
complete -c fsys -f
complete -c ffs -f

# Options and positional command/scope candidates are valid before the command.
set -l __fsys_cond 'not __fsys_seen_command'

# Global options.
complete -c fsys -n "$__fsys_cond" -a '-u'          -d 'Use the per-user systemd manager'
complete -c fsys -n "$__fsys_cond" -a '--user'       -d 'Use the per-user systemd manager'
complete -c fsys -n "$__fsys_cond" -a '-s'          -d 'Use the system manager'
complete -c fsys -n "$__fsys_cond" -a '--system'     -d 'Use the system manager'
complete -c fsys -n "$__fsys_cond" -a '-y'          -d 'Bypass mutation confirmations'
complete -c fsys -n "$__fsys_cond" -a '--yes'       -d 'Bypass mutation confirmations'
complete -c fsys -n "$__fsys_cond" -a '--no-confirm' -d 'Bypass mutation confirmations'
complete -c fsys -n "$__fsys_cond" -a '-h'          -d 'Show help'
complete -c fsys -n "$__fsys_cond" -a '--help'       -d 'Show help'
complete -c fsys -n "$__fsys_cond" -a '--version'    -d 'Print the fsys version'
complete -c fsys -n "$__fsys_cond" -a '--'          -d 'End global option parsing'

complete -c ffs -n "$__fsys_cond" -a '-u'          -d 'Use the per-user systemd manager'
complete -c ffs -n "$__fsys_cond" -a '--user'       -d 'Use the per-user systemd manager'
complete -c ffs -n "$__fsys_cond" -a '-s'          -d 'Use the system manager'
complete -c ffs -n "$__fsys_cond" -a '--system'     -d 'Use the system manager'
complete -c ffs -n "$__fsys_cond" -a '-y'          -d 'Bypass mutation confirmations'
complete -c ffs -n "$__fsys_cond" -a '--yes'       -d 'Bypass mutation confirmations'
complete -c ffs -n "$__fsys_cond" -a '--no-confirm' -d 'Bypass mutation confirmations'
complete -c ffs -n "$__fsys_cond" -a '-h'          -d 'Show help'
complete -c ffs -n "$__fsys_cond" -a '--help'       -d 'Show help'
complete -c ffs -n "$__fsys_cond" -a '--version'    -d 'Print the fsys version'
complete -c ffs -n "$__fsys_cond" -a '--'          -d 'End global option parsing'

# Explicit manager scopes.
complete -c fsys -n "$__fsys_cond" -a 'system' -d 'Use the system manager'
complete -c fsys -n "$__fsys_cond" -a 'user'   -d 'Use the per-user systemd manager'
complete -c ffs  -n "$__fsys_cond" -a 'system' -d 'Use the system manager'
complete -c ffs  -n "$__fsys_cond" -a 'user'   -d 'Use the per-user systemd manager'

# Commands and aliases.
complete -c fsys -n "$__fsys_cond" -a 'service'    -d 'Browse system services'
complete -c fsys -n "$__fsys_cond" -a 'services'   -d 'Browse system services'
complete -c fsys -n "$__fsys_cond" -a 'svc'        -d 'Alias for services'
complete -c fsys -n "$__fsys_cond" -a 'timer'      -d 'Browse system timers'
complete -c fsys -n "$__fsys_cond" -a 'timers'     -d 'Browse system timers'
complete -c fsys -n "$__fsys_cond" -a 'unit'       -d 'Browse loaded system units'
complete -c fsys -n "$__fsys_cond" -a 'units'      -d 'Browse loaded system units'
complete -c fsys -n "$__fsys_cond" -a 'unit-file'  -d 'Browse installed system unit files'
complete -c fsys -n "$__fsys_cond" -a 'unit-files' -d 'Browse installed system unit files'
complete -c fsys -n "$__fsys_cond" -a 'failed'     -d 'Browse failed system units'
complete -c fsys -n "$__fsys_cond" -a 'log'        -d 'Browse journal sources by unit'
complete -c fsys -n "$__fsys_cond" -a 'logs'       -d 'Browse journal sources by unit'
complete -c fsys -n "$__fsys_cond" -a 'journal'    -d 'Alias for logs'
complete -c fsys -n "$__fsys_cond" -a 'boot'       -d 'Browse journal boots'
complete -c fsys -n "$__fsys_cond" -a 'boots'      -d 'Alias for boot'
complete -c fsys -n "$__fsys_cond" -a 'overview'   -d 'Show system-manager overview'
complete -c fsys -n "$__fsys_cond" -a 'status'     -d 'Alias for overview'
complete -c fsys -n "$__fsys_cond" -a 'linger'     -d 'Manage linger for the current user'

complete -c ffs -n "$__fsys_cond" -a 'service'    -d 'Browse system services'
complete -c ffs -n "$__fsys_cond" -a 'services'   -d 'Browse system services'
complete -c ffs -n "$__fsys_cond" -a 'svc'        -d 'Alias for services'
complete -c ffs -n "$__fsys_cond" -a 'timer'      -d 'Browse system timers'
complete -c ffs -n "$__fsys_cond" -a 'timers'     -d 'Browse system timers'
complete -c ffs -n "$__fsys_cond" -a 'unit'       -d 'Browse loaded system units'
complete -c ffs -n "$__fsys_cond" -a 'units'      -d 'Browse loaded system units'
complete -c ffs -n "$__fsys_cond" -a 'unit-file'  -d 'Browse installed system unit files'
complete -c ffs -n "$__fsys_cond" -a 'unit-files' -d 'Browse installed system unit files'
complete -c ffs -n "$__fsys_cond" -a 'failed'     -d 'Browse failed system units'
complete -c ffs -n "$__fsys_cond" -a 'log'        -d 'Browse journal sources by unit'
complete -c ffs -n "$__fsys_cond" -a 'logs'       -d 'Browse journal sources by unit'
complete -c ffs -n "$__fsys_cond" -a 'journal'    -d 'Alias for logs'
complete -c ffs -n "$__fsys_cond" -a 'boot'       -d 'Browse journal boots'
complete -c ffs -n "$__fsys_cond" -a 'boots'      -d 'Alias for boot'
complete -c ffs -n "$__fsys_cond" -a 'overview'   -d 'Show system-manager overview'
complete -c ffs -n "$__fsys_cond" -a 'status'     -d 'Alias for overview'
complete -c ffs -n "$__fsys_cond" -a 'linger'     -d 'Manage linger for the current user'

# After a command has been selected, remaining words are free-form fzf query text.
complete -c fsys -n '__fsys_seen_command' -f
complete -c ffs  -n '__fsys_seen_command' -f
