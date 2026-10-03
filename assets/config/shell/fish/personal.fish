


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
set -gx STARSHIP_CONFIG "$UNISHELL_CONFIG_PATH/starship/personal.toml"

# ╚═══════════════════════════════════════════════════════════════
# vim: set ft=fish:
