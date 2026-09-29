# Fish completion for fdoc and the optional `ffd` alias/function.
#
# Install (user-wide):
#   mkdir -p ~/.config/fish/completions
#   cp fdoc.fish ~/.config/fish/completions/fdoc.fish
#
# `ffd` inherits the same completion set through Fish's native wrapping.
#
# Important design points:
#   * Ordinary filename completion is disabled globally for fdoc. Without this,
#     Fish can fall back to listing the current directory when no custom
#     completion applies.
#   * File/directory completion is explicitly re-enabled only for Compose path
#     options.
#   * The first positional argument is treated as fdoc's resource selector.
#     Global -c/--context may appear before that selector, and resource-specific
#     options may appear after it.
#   * The parser intentionally ignores option arguments while deciding whether a
#     resource has already been selected, so `fdoc -c prod <TAB>` still offers
#     resource names rather than treating `prod` as a resource.

# -----------------------------------------------------------------------------
# Dynamic helpers
# -----------------------------------------------------------------------------

function __fdoc_contexts
    command -q docker; or return
    command docker context ls --format '{{.Name}}' 2>/dev/null
end

function __fdoc_resources
    printf '%s\t%s\n' \
        containers 'Browse containers' \
        container 'Alias for containers' \
        ps 'Alias for containers' \
        images 'Browse images' \
        image 'Alias for images' \
        img 'Alias for images' \
        volumes 'Browse Docker volumes' \
        volume 'Alias for volumes' \
        vol 'Alias for volumes' \
        networks 'Browse Docker networks' \
        network 'Alias for networks' \
        net 'Alias for networks' \
        compose 'Browse Compose services' \
        composes 'Alias for compose' \
        services 'Alias for compose' \
        service 'Alias for compose' \
        ctx 'Browse and switch Docker contexts' \
        context 'Alias for ctx' \
        contexts 'Alias for ctx' \
        resource 'Browse resource types' \
        resources 'Alias for resource' \
        type 'Alias for resource' \
        types 'Alias for resource' \
        help 'Show fdoc help'
end

# Return success when a real fdoc resource has already appeared in the
# commandline before the token currently being completed.
#
# commandline -opc returns the command and all tokens before the current token.
# See: https://fishshell.com/docs/current/cmds/commandline.html
function __fdoc_has_resource
    set -l tokens (commandline -opc)
    set -l count (count $tokens)

    # The command name itself is token 1. Nothing after it means there is no
    # selected resource yet.
    test $count -gt 1; or return 1

    set -l skip_next 0

    for token in $tokens[2..-1]
        if test $skip_next -eq 1
            set skip_next 0
            continue
        end

        switch $token
            case '-c' '--context'
                # The next token is the context value, not the resource.
                set skip_next 1
                continue

            case '-c=*' '--context=*'
                # Attached context forms: -c=name / --context=name.
                continue

            case '-c?*'
                # Attached short form: -cname. `-c` itself was handled above.
                continue

            case '--'
                # fdoc treats everything after -- as query text. At this point
                # there is still no resource selector.
                return 1

            case '-*'
                # Other leading-dash tokens are options. None of these can be a
                # resource selector.
                continue

            case '*'
                # First non-option, non-context-value token is the resource.
                return 0
        end
    end

    return 1
end

function __fdoc_resource
    set -l tokens (commandline -opc)
    set -l count (count $tokens)
    test $count -gt 1; or return

    set -l skip_next 0

    for token in $tokens[2..-1]
        if test $skip_next -eq 1
            set skip_next 0
            continue
        end

        switch $token
            case '-c' '--context'
                set skip_next 1
                continue
            case '-c=*' '--context=*'
                continue
            case '-c?*'
                continue
            case '--'
                return
            case '-*'
                return
            case '*'
                printf '%s\n' $token
                return
        end
    end
end

function __fdoc_resource_is
    set -l wanted $argv[1]
    set -l resource (__fdoc_resource)

    switch $resource
        case containers container ps
            test $wanted = containers
        case images image img
            test $wanted = images
        case volumes volume vol
            test $wanted = volumes
        case networks network net
            test $wanted = networks
        case compose composes services service
            test $wanted = compose
        case ctx context contexts
            test $wanted = ctx
        case resource resources type types
            test $wanted = resource
        case '*'
            return 1
    end
end

# -----------------------------------------------------------------------------
# Base completion behavior
# -----------------------------------------------------------------------------

# Do not let Fish fall back to ordinary directory/file completion for fdoc.
complete -c fdoc -f

# Offer the resource selector whenever one has not already been selected.
# This is intentionally independent of the current partial token, so Fish can
# perform its own prefix/infix filtering.
complete -c fdoc \
    -n 'not __fdoc_has_resource' \
    -a '(__fdoc_resources)'

# -----------------------------------------------------------------------------
# Global options
# -----------------------------------------------------------------------------

complete -c fdoc \
    -s c -l context -r \
    -a '(__fdoc_contexts)' \
    -d 'Use a Docker context for this invocation'

complete -c fdoc \
    -s h -l help \
    -d 'Show fdoc help'

complete -c fdoc \
    -s V -l version \
    -d 'Print the fdoc version'

# -----------------------------------------------------------------------------
# Engine resources
# -----------------------------------------------------------------------------

complete -c fdoc \
    -n '__fdoc_resource_is containers; or __fdoc_resource_is images; or __fdoc_resource_is volumes; or __fdoc_resource_is networks' \
    -s h -l help \
    -d 'Show fdoc help'

# -----------------------------------------------------------------------------
# Compose options
# -----------------------------------------------------------------------------

# Compose path options. -F deliberately restores file completion for these
# options despite the global `complete -c fdoc -f` above.
complete -c fdoc \
    -n '__fdoc_resource_is compose' \
    -s f -l file -r -F \
    -a '(__fish_complete_path (commandline -ct) "Compose file")' \
    -d 'Compose file (repeatable)'

complete -c fdoc \
    -n '__fdoc_resource_is compose' \
    -s p -l project-name -r \
    -d 'Compose project name'

complete -c fdoc \
    -n '__fdoc_resource_is compose' \
    -l project-directory -r -F \
    -a '(__fish_complete_directories (commandline -ct) "Compose project directory")' \
    -d 'Compose project directory'

complete -c fdoc \
    -n '__fdoc_resource_is compose' \
    -l profile -r \
    -d 'Compose profile (repeatable)'

complete -c fdoc \
    -n '__fdoc_resource_is compose' \
    -l env-file -r -F \
    -a '(__fish_complete_path (commandline -ct) "Compose environment file")' \
    -d 'Compose environment file (repeatable)'

complete -c fdoc \
    -n '__fdoc_resource_is compose' \
    -s h -l help \
    -d 'Show fdoc help'

# -----------------------------------------------------------------------------
# Context / resource-picker modes
# -----------------------------------------------------------------------------

complete -c fdoc \
    -n '__fdoc_resource_is ctx' \
    -s h -l help \
    -d 'Show fdoc help'

complete -c fdoc \
    -n '__fdoc_resource_is resource' \
    -s h -l help \
    -d 'Show fdoc help'

# -----------------------------------------------------------------------------
# ffd: inherit the complete fdoc definition.
# -----------------------------------------------------------------------------

complete -c ffd -w fdoc

# vim: set ft=fish:
