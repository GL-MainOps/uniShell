#compdef unishell

_unishell() {
    local -a commands
    commands=(
        'shell:Start or attach to the uniShell environment'
        'install:Install persistent uniShell'
        'update:Update the installed runtime'
        'upgrade:Alias for update'
        'clean:Clean sessions or the installed runtime'
        'list:List managed sessions'
        'detach:Detach from a multiplexer session'
        'version:Display version information'
        'help:Display help'
        'completion:Print shell completion script'
    )

    if (( CURRENT == 2 )); then
        _describe 'uniShell command' commands
        return
    fi

    case "${words[2]}" in
        completion)
            _values 'shell' bash zsh fish
            return
            ;;
        clean)
            _arguments \
                '--target=[Clean a named session]:session name:' \
                '--installed[Remove the complete persistent runtime after confirmation]' \
                && return
            ;;
    esac

    _arguments -C \
        '--runtime-dir=[Set runtime root]:directory:_files -/' \
        '--shell=[Select shell]:shell:(bash zsh fish nushell)' \
        '--shell-profile=[Select shell profile]:profile:' \
        '--multiplexer=[Select multiplexer]:multiplexer:(tmux zellij none disabled)' \
        '--session=[Set session name]:name:' \
        '--multiplexer-session=[Set native multiplexer session name]:name:' \
        '--new-session[Start a new multiplexer session]' \
        '--no-shared-rc[Skip shared shell configuration]' \
        '--shared-rc[Load shared shell configuration]'
}

_unishell "$@"
