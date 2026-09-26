complete -c unishell -f

complete -c unishell -n '__fish_use_subcommand' -a 'shell install update upgrade clean list detach version help completion'
complete -c unishell -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
complete -c unishell -n '__fish_seen_subcommand_from clean' -l target -r -d 'Clean the named session'
complete -c unishell -n '__fish_seen_subcommand_from clean' -l installed -d 'Remove the persistent runtime after confirmation'

complete -c unishell -l runtime-dir -r -d 'Set the runtime root'
complete -c unishell -l shell -r -a 'bash zsh fish nushell' -d 'Select a shell'
complete -c unishell -l shell-profile -r -d 'Select a shell profile'
complete -c unishell -l multiplexer -r -a 'tmux zellij none disabled' -d 'Select a multiplexer'
complete -c unishell -l session -r -d 'Set the session name'
complete -c unishell -l multiplexer-session -r -d 'Set the native multiplexer session name'
complete -c unishell -l new-session -d 'Start a new multiplexer session'
complete -c unishell -l no-shared-rc -d 'Skip shared shell configuration'
complete -c unishell -l shared-rc -d 'Load shared shell configuration'
# vim: set ft=fish:
