


# ╔═══════════════════════════════════════════════════════════════
# ║ SOURCING SHARED CONFIGURATIONS

source "$UNISHELL_CONFIG_SHELL_PATH/_cross_profiles_config/shared.bash"

# ╚═══════════════════════════════════════════════════════════════



# ╔═══════════════════════════════════════════════════════════════
# ║ PROFILE-SPECIFIC CONFIGURATIONS

# ╭────────────────────────────────╮
# │ rcopy-RELATED CONFIG           │
# ╰────────────────────────────────╯

# ┌────────────────┐
# │ ADD rcopy TO   │
# │ PATH           │
# └────────────────┘
if [[ ":$PATH:" != *":$UNISHELL_SESSION_RUNTIME_DIR/scripts/rcopy:"* ]]; then
    export PATH="$PATH:$UNISHELL_SESSION_RUNTIME_DIR/scripts/rcopy"
fi

# ┌────────────────┐
# │ PRUNE STALE    │
# │ SOCKETS        │
# └────────────────┘
rcopy --prune

# ╚═══════════════════════════════════════════════════════════════
# vim: set ft=bash:
