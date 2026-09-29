# shellcheck shell=bash
#
# Bash completion for fkube.
#
# Installs completion for both `fkube` and the intended `ffk` alias:
#
#   source /path/to/fkube.bash
#
# or place this file somewhere loaded by bash-completion, for example:
#   ~/.local/share/bash-completion/completions/fkube
#
# The completion is intentionally self-contained. It does not require the
# fkube implementation itself to have been sourced before this file.
#
# Dynamic completion sources:
#   - resource names and short names: kubectl api-resources
#   - namespaces: kubectl get namespaces
#   - contexts: kubectl config get-contexts
#
# The live Kubernetes queries are best-effort. If kubectl is unavailable or a
# query fails, static fkube options/subcommands still complete normally.

_fkube_bash__unique_words() {
    # Read newline-separated words and emit each only once.
    awk '!seen[$0]++ && $0 != "" { print }'
}

_fkube_bash__resources() {
    command kubectl api-resources --verbs=list --namespaced=true 2>/dev/null
    command kubectl api-resources --verbs=list --namespaced=false 2>/dev/null
}

_fkube_bash__resource_names() {
    _fkube_bash__resources |
        awk '
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
        ' |
        _fkube_bash__unique_words
}

_fkube_bash__namespaces() {
    command kubectl get namespaces -o name 2>/dev/null |
        sed 's#^namespace/##' |
        _fkube_bash__unique_words
}

_fkube_bash__contexts() {
    command kubectl config get-contexts -o name 2>/dev/null |
        _fkube_bash__unique_words
}

_fkube_bash__append_matches() {
    local cur=$1
    shift
    local item
    COMPREPLY=()
    while IFS= read -r item; do
        [[ $item == "$cur"* ]] && COMPREPLY+=("$item")
    done < <(printf '%s\n' "$@")
}

_fkube_bash__complete_words() {
    local cur=$1
    shift
    local wordlist

    # Kubernetes names and fkube options do not contain whitespace, so a
    # space-separated compgen word list is both simple and compatible with
    # Bash's normal completion escaping.
    printf -v wordlist '%s ' "$@"
    COMPREPLY=( $(compgen -W "$wordlist" -- "$cur") )
}

_fkube_bash__complete() {
    local cur prev word
    local cword=$COMP_CWORD
    local -a words=()
    local -a static=()
    local -a values=()
    local i
    local namespace_expected=0
    local skip_next=0
    local seen_subcommand=''
    local current_command=''
    local has_primary=0

    # COMP_WORDS is already shell-tokenized by Bash, which is exactly what we
    # want here. Copy it so the rest of the function can use straightforward
    # positional inspection without mutating Bash's completion globals.
    words=("${COMP_WORDS[@]}")
    cur=${COMP_WORDS[COMP_CWORD]-}
    prev=${COMP_WORDS[COMP_CWORD-1]-}

    # We intentionally treat fkube's first non-option command as the main
    # subcommand/resource. Global safety and scope flags may appear before or
    # after it (until `--`), matching fkube's own parser.
    for ((i=1; i<cword; i++)); do
        word=${words[i]}

        if ((skip_next)); then
            skip_next=0
            namespace_expected=0
            continue
        fi

        case $word in
            -n|--namespace)
                namespace_expected=1
                skip_next=1
                continue
                ;;
            -A|--all-namespaces|-y|--yes|--assume-yes|--confirm|--no-confirm)
                namespace_expected=0
                continue
                ;;
            --)
                namespace_expected=0
                break
                ;;
            ns|namespace|namespaces|ctx|context|contexts|resource|resources|type|types|doctor|--doctor|help|-h|--help|version|--version)
                [[ -z $seen_subcommand ]] && seen_subcommand=$word
                has_primary=1
                namespace_expected=0
                continue
                ;;
            *)
                # The first unknown non-option token is a Kubernetes resource
                # name (or shorthand). Keep parsing flags after it, because
                # `fkube pods -n NAMESPACE` is valid.
                if [[ -z $current_command ]]; then
                    current_command=$word
                fi
                has_primary=1
                namespace_expected=0
                ;;
        esac
    done

    # `-n`/`--namespace` always takes a namespace next. This is checked first
    # so it wins over resource/subcommand suggestions.
    if ((namespace_expected)) || [[ $prev == -n || $prev == --namespace ]]; then
        values=( $(_fkube_bash__namespaces) )
        _fkube_bash__complete_words "$cur" "${values[*]}"
        return 0
    fi

    static=(
        'pods'
        'deploy'
        'svc'
        'ds'
        'sts'
        'rs'
        'no'
        'ns'
        'namespace'
        'namespaces'
        'ctx'
        'context'
        'contexts'
        'resource'
        'resources'
        'type'
        'types'
        'doctor'
        'help'
        'version'
        '--doctor'
        '--help'
        '--version'
        '-h'
        '-n'
        '--namespace'
        '-A'
        '--all-namespaces'
        '-y'
        '--yes'
        '--assume-yes'
        '--no-confirm'
        '--confirm'
    )

    # Completing the first argument: include both convenience aliases and the
    # complete live resource/short-name set from Kubernetes.
    if ((cword == 1)) || [[ $has_primary -eq 0 ]]; then
        values=( "${static[@]}" )
        while IFS= read -r word; do values+=("$word"); done < <(_fkube_bash__resource_names)
        _fkube_bash__complete_words "$cur" "${values[*]}"
        return 0
    fi

    case $seen_subcommand in
        ns|namespace|namespaces)
            # Namespace gate accepts arbitrary fzf query words, but suggesting
            # existing namespaces makes the common case much faster.
            values=( $(_fkube_bash__namespaces) )
            _fkube_bash__complete_words "$cur" "${values[*]}"
            return 0
            ;;
        ctx|context|contexts)
            values=( $(_fkube_bash__contexts) )
            _fkube_bash__complete_words "$cur" "${values[*]}"
            return 0
            ;;
        resource|resources|type|types)
            values=( $(_fkube_bash__resource_names) )
            _fkube_bash__complete_words "$cur" "${values[*]}"
            return 0
            ;;
        doctor|--doctor|help|-h|--help|version|--version)
            # These commands do not have positional arguments of their own.
            COMPREPLY=()
            return 0
            ;;
    esac

    # A resource is selected at this point. Options that are meaningful for the
    # resource picker can still be completed after it.
    case $cur in
        -n|--namespace|-A|--all-namespaces|-y|--yes|--assume-yes|--confirm|--no-confirm|--help|--version)
            # This branch lets compgen use the normal prefix matching below.
            ;;
    esac

    # Do not offer the namespace/context/namespace-gate aliases as new
    # positional arguments after a resource; the remaining words are fzf query
    # terms, so leaving them unconstrained is deliberate.
    _fkube_bash__complete_words "$cur" \
        '-n --namespace -A --all-namespaces -y --yes --assume-yes --no-confirm --confirm -h --help --version'
    return 0
}

# `complete` accepts multiple command names, so one completion function covers
# both the real command and the user's intended `ffk` alias.
complete -F _fkube_bash__complete fkube ffk
