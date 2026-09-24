_unishell_complete() {
    local cur prev command
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    command="${COMP_WORDS[1]}"

    case "$prev" in
        --runtime-dir|--session|--multiplexer-session)
            return
            ;;
        --shell)
            COMPREPLY=( $(compgen -W "bash zsh fish nushell" -- "$cur") )
            return
            ;;
        --shell-profile)
            return
            ;;
        --multiplexer)
            COMPREPLY=( $(compgen -W "tmux zellij none disabled" -- "$cur") )
            return
            ;;
        --target)
            return
            ;;
    esac

    if [[ "$command" == "completion" ]]; then
        if (( COMP_CWORD == 2 )); then
            COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
        fi
        return
    fi

    if [[ "$command" == "clean" ]]; then
        COMPREPLY=( $(compgen -W "--target --installed" -- "$cur") )
        return
    fi

    local flags="--runtime-dir --shell --shell-profile --multiplexer --session --multiplexer-session --new-session --no-shared-rc --shared-rc"
    if (( COMP_CWORD == 1 )); then
        local commands="shell install update upgrade clean list detach version help completion"
        COMPREPLY=( $(compgen -W "$commands $flags" -- "$cur") )
    elif [[ "$command" == "shell" ]]; then
        COMPREPLY=( $(compgen -W "$flags" -- "$cur") )
    fi
}

complete -F _unishell_complete unishell
