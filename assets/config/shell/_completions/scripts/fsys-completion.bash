# Bash completion for fsys
#
# Install, for example:
#   source /path/to/fsys.bash
#
# This completion intentionally targets both:
#   fsys
#   ffs
#
# The fsys command treats options as leading/global options, followed by an
# optional explicit scope (system/user), then a command, then an arbitrary fzf
# query. Once fsys is inside the free-form query portion, completion stops.

_fsys_complete() {
    local cur prev word
    local -a commands command_aliases global_opts scopes scope_opts
    local command='' scope='' i
    local -a words_before_command=()

    COMPREPLY=()
    cur=${COMP_WORDS[COMP_CWORD]-}
    prev=${COMP_WORDS[COMP_CWORD-1]-}

    commands=(
        service
        services
        svc
        timer
        timers
        unit
        units
        unit-file
        unit-files
        failed
        log
        logs
        journal
        boot
        boots
        overview
        status
        linger
    )

    global_opts=(
        '-u'
        '--user'
        '-s'
        '--system'
        '-y'
        '--yes'
        '--no-confirm'
        '-h'
        '--help'
        '--version'
        '--'
    )

    scopes=(
        system
        user
    )

    scope_opts=(
        '-u'
        '--user'
        '-s'
        '--system'
    )

    # Find the first command token, honoring the parser's rule that options and
    # scopes may appear before it. Everything after that is an fzf query and
    # should be left to normal Bash completion.
    for ((i=1; i<COMP_CWORD; i++)); do
        word=${COMP_WORDS[i]}

        if [[ -z $command ]]; then
            case $word in
                --)
                    # `--` ends option parsing; the next token is treated as
                    # the command by fsys, so do not complete options after it.
                    if (( i + 1 < COMP_CWORD )); then
                        command=${COMP_WORDS[i+1]}
                    fi
                    break
                    ;;
                -u|--user|-s|--system)
                    scope=${word#--}
                    [[ $word == -u ]] && scope=user
                    [[ $word == -s ]] && scope=system
                    continue
                    ;;
                -y|--yes|--no-confirm|-h|--help|--version)
                    continue
                    ;;
                system|user)
                    scope=$word
                    continue
                    ;;
                service|services|svc|timer|timers|unit|units|unit-file|unit-files|failed|log|logs|journal|boot|boots|overview|status|linger)
                    command=$word
                    break
                    ;;
                -*)
                    # Unknown option: let Bash's normal fallback handle it.
                    return 0
                    ;;
                *)
                    # First non-option/non-scope token is the command.
                    command=$word
                    break
                    ;;
            esac
        fi
    done

    # Current word can itself be the command being completed.
    if [[ -z $command ]]; then
        case $cur in
            -*)
                COMPREPLY=(
                    $(compgen -W "${global_opts[*]}" -- "$cur")
                )
                return 0
                ;;
            *)
                COMPREPLY=(
                    $(compgen -W "${commands[*]} ${scopes[*]}" -- "$cur")
                )
                return 0
                ;;
        esac
    fi

    # If the previous tokens have established a command, this script has no
    # structured positional arguments after it: the remainder is an fzf query.
    # Offer no noisy filesystem/option completions.
    return 0
}

# Register completion for both the real command/function and the intended alias.
complete -F _fsys_complete fsys
complete -F _fsys_complete ffs
