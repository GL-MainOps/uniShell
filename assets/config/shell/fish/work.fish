


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
