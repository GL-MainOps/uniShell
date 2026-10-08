


# ╔═══════════════════════════════════════════════════════════════
# ║ SOURCING SHARED CONFIGURATIONS

source "$UNISHELL_CONFIG_SHELL_PATH/_cross_profiles_config/shared.fish"

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ PROFILE-SPECIFIC CONFIGURATIONS

# ╭────────────────────────────────╮
# │ VARIABLES                      │
# ╰────────────────────────────────╯
# ┌────────────────┐
# │ STARSHIP       │
# └────────────────┘
set -gx STARSHIP_CONFIG "$UNISHELL_CONFIG_PATH/starship/work.toml"

# function starship_transient_prompt_func
#   starship module line_break
#   starship module custom.timestamp
#   starship module character
#   starship module line_break
# end

# function starship_transient_rprompt_func
# end

# ╭────────────────────────────────╮
# │ rcopy-RELATED CONFIG           │
# ╰────────────────────────────────╯

# ┌────────────────┐
# │ ADD rcopy TO   │
# │ PATH           │
# └────────────────┘
# fish_add_path is idempotent, it the path already available, it will not append it
set -gx PATH "$UNISHELL_SESSION_RUNTIME_DIR/scripts/rcopy" $PATH

# ┌────────────────┐
# │ PRUNE STALE    │
# │ SOCKETS        │
# └────────────────┘
rcopy --prune

# ╚═══════════════════════════════════════════════════════════════
# vim: set ft=fish:
