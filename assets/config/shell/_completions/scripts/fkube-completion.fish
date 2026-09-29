# fkube - Fish completion
#
# Installs completion for both `fkube` and the intended `ffk` alias.
#
# This file is self-contained and dynamically completes:
#   - Kubernetes resources and short names via `kubectl api-resources`
#   - namespaces via `kubectl get namespaces`
#   - kubeconfig contexts via `kubectl config get-contexts`
#
# Install manually with:
#   cp fkube.fish ~/.config/fish/completions/fkube.fish
#
# The completion definitions are registered for both command names because a
# Fish alias/function named `ffk` does not automatically inherit `fkube`'s
# completion definitions.

function _fkube_fish_resources
    command kubectl api-resources --verbs=list --namespaced=true 2>/dev/null
    command kubectl api-resources --verbs=list --namespaced=false 2>/dev/null
end

function _fkube_fish_resource_names
    _fkube_fish_resources | awk '
        function emit_shortnames(value,    n, a, i) {
            if (value == "" || value == "<none>") return
            n = split(value, a, ",")
            for (i = 1; i <= n; i++)
                if (a[i] != "") print a[i]
        }

        /^NAME[[:space:]]+SHORTNAMES([[:space:]]|$)/ {
            name_col = short_col = 0
            for (i = 1; i <= NF; i++) {
                if ($i == "NAME") name_col = i
                else if ($i == "SHORTNAMES") short_col = i
            }
            next
        }

        name_col {
            print $name_col
            if (short_col) emit_shortnames($short_col)
        }
    ' | sort -u
end

function _fkube_fish_namespaces
    command kubectl get namespaces -o name 2>/dev/null |
        string replace -r '^namespace/' '' |
        sort -u
end

function _fkube_fish_contexts
    command kubectl config get-contexts -o name 2>/dev/null | sort -u
end

function _fkube_fish_previous_is_namespace_option
    set -l tokens (commandline -opc)
    test (count $tokens) -gt 1
    or return 1

    set -l previous $tokens[-1]
    test "$previous" = -n -o "$previous" = --namespace
end

function _fkube_fish_seen_subcommand
    set -l wanted $argv[1..]
    set -l tokens (commandline -opc)
    test (count $tokens) -gt 1
    or return 1

    for token in $tokens[2..-1]
        for candidate in $wanted
            if test "$token" = "$candidate"
                return 0
            end
        end
    end
    return 1
end

function _fkube_fish_complete_namespaces
    _fkube_fish_namespaces
end

function _fkube_fish_complete_contexts
    _fkube_fish_contexts
end

function _fkube_fish_complete_resources
    _fkube_fish_resource_names
end

# -----------------------------------------------------------------------------
# First command/resource position
# -----------------------------------------------------------------------------

set -l _fkube_subcommands \
    'pods' \
    'deploy' \
    'svc' \
    'ds' \
    'sts' \
    'rs' \
    'no' \
    'ns' \
    'namespace' \
    'namespaces' \
    'ctx' \
    'context' \
    'contexts' \
    'resource' \
    'resources' \
    'type' \
    'types' \
    'doctor' \
    'help' \
    'version'

# Static aliases/subcommands. The dynamic resource list is added separately.
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'pods' -d 'Browse pods'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'deploy' -d 'Browse deployments'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'svc' -d 'Browse services'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'ds' -d 'Browse daemonsets'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'sts' -d 'Browse statefulsets'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'rs' -d 'Browse replicasets'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'no' -d 'Browse nodes'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'ns namespace namespaces' -d 'Choose a namespace'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'ctx context contexts' -d 'Choose/switch kubeconfig context'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'resource resources type types' -d 'Choose a Kubernetes resource type'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'doctor' -d 'Check fkube prerequisites'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'help' -d 'Show fkube help'
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a 'version' -d 'Show fkube version'

# Dynamic Kubernetes resources and short names at the first position.
complete -c fkube -c ffk -f -n '__fish_use_subcommand' -a '(_fkube_fish_complete_resources)' -d 'Kubernetes resource / short name'

# -----------------------------------------------------------------------------
# Global options
# -----------------------------------------------------------------------------

complete -c fkube -c ffk -f -s n -l namespace -r -a '(_fkube_fish_complete_namespaces)' -d 'Starting namespace'
complete -c fkube -c ffk -f -s A -l all-namespaces -d 'Start in all namespaces'
complete -c fkube -c ffk -f -s y -l yes -l assume-yes -d 'Skip mutation confirmations'
complete -c fkube -c ffk -f -l no-confirm -d 'Disable mutation confirmations'
complete -c fkube -c ffk -f -l confirm -d 'Force mutation confirmations'
complete -c fkube -c ffk -f -s h -l help -d 'Show help'
complete -c fkube -c ffk -f -l version -d 'Show version'
complete -c fkube -c ffk -f -l doctor -d 'Run prerequisite diagnostics'

# -----------------------------------------------------------------------------
# Namespace/context/resource query completion
# -----------------------------------------------------------------------------

# `fkube ns ...`: arbitrary text is accepted as an fzf query; offering real
# namespace names makes common queries convenient without restricting input.
complete -c fkube -c ffk -f \
    -n '_fkube_fish_seen_subcommand ns namespace namespaces' \
    -a '(_fkube_fish_complete_namespaces)' \
    -d 'Namespace'

# `fkube ctx ...`: same idea for kubeconfig contexts.
complete -c fkube -c ffk -f \
    -n '_fkube_fish_seen_subcommand ctx context contexts' \
    -a '(_fkube_fish_complete_contexts)' \
    -d 'Kubeconfig context'

# `fkube resource ...`: choose a resource type directly.
complete -c fkube -c ffk -f \
    -n '_fkube_fish_seen_subcommand resource resources type types' \
    -a '(_fkube_fish_complete_resources)' \
    -d 'Kubernetes resource / short name'

# Explicitly expose the common resource options after a selected resource.
# Fish's normal completion engine already handles the argument to -n/--namespace
# using the declaration above.
complete -c fkube -c ffk -f -n 'not __fish_seen_subcommand_from ns namespace namespaces ctx context contexts' \
    -a '-A --all-namespaces -y --yes --assume-yes --no-confirm --confirm -h --help --version --doctor' \
    -d 'fkube option'
