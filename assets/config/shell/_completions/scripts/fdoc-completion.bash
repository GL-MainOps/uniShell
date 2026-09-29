# bash-completion for fdoc and the optional `ffd` alias.
#
# Install (user-wide):
#   mkdir -p ~/.local/share/bash-completion/completions
#   cp fdoc.bash ~/.local/share/bash-completion/completions/fdoc
#
# Or source this file from ~/.bashrc:
#   source /path/to/fdoc.bash
#
# The completion is intentionally independent of the fdoc implementation, so
# it remains useful when fdoc itself is provided as a sourced Bash function.
# Both command names are completed:
#   fdoc ...
#   ffd ...        # e.g. alias ffd='fdoc'

_fdoc_complete__contexts() {
    local contexts

    command -v docker >/dev/null 2>&1 || return 0
    contexts=$(docker context ls --format '{{.Name}}' 2>/dev/null) || return 0

    printf '%s\n' "$contexts"
}

_fdoc_complete__files() {
    local cur=${1-}
    compgen -f -- "$cur" 2>/dev/null
}

_fdoc_complete__dirs() {
    local cur=${1-}
    compgen -d -- "$cur" 2>/dev/null
}

_fdoc_complete__contexts_reply() {
    local cur=${1-}
    local context
    local -a matches=()

    while IFS= read -r context; do
        [[ -n $context ]] || continue
        [[ $context == "$cur"* ]] && matches+=("$context")
    done < <(_fdoc_complete__contexts)

    COMPREPLY=("${matches[@]}")
}

_fdoc_complete__values_reply() {
    local cur=${1-}
    shift
    local value
    local -a matches=()

    for value in "$@"; do
        [[ $value == "$cur"* ]] && matches+=("$value")
    done

    COMPREPLY=("${matches[@]}")
}

_fdoc_complete__resource() {
    local resource=${1-}

    case $resource in
        containers|container|ps) printf '%s' containers ;;
        images|image|img)      printf '%s' images ;;
        volumes|volume|vol)    printf '%s' volumes ;;
        networks|network|net)  printf '%s' networks ;;
        compose|composes|services|service) printf '%s' compose ;;
        ctx|context|contexts)  printf '%s' ctx ;;
        resource|resources|type|types) printf '%s' resource ;;
        help) printf '%s' help ;;
        *) return 1 ;;
    esac
}

_fdoc_complete__find_resource() {
    # Print the first real top-level resource token before the current word.
    # Global context options may occur before it and consume one value.
    local -n _words=$1
    local cword=$2
    local i=1
    local token

    while ((i < cword)); do
        token=${_words[i]}
        case $token in
            -c|--context)
                ((i += 2))
                continue
                ;;
            --context=*|-c=*)
                ((i++))
                continue
                ;;
            -c?*)
                ((i++))
                continue
                ;;
            --help|-h|--version|-V)
                ((i++))
                continue
                ;;
            --)
                # `fdoc -- ...` is not a documented way to select a resource;
                # the implementation treats the remaining text as a query.
                return 1
                ;;
            -*)
                return 1
                ;;
            *)
                printf '%s' "$token"
                return 0
                ;;
        esac
    done

    return 1
}

_fdoc_complete() {
    local cur=${COMP_WORDS[COMP_CWORD]}
    local prev=${COMP_WORDS[COMP_CWORD-1]-}
    local resource raw_resource normalized_resource
    local -a words=()

    COMPREPLY=()
    local COMP_WORDBREAKS=${COMP_WORDBREAKS//:/}

    # Preserve Bash's own handling of an empty command line while avoiding
    # unwanted filename suggestions for fdoc's free-form fzf search queries.
    words=("${COMP_WORDS[@]}")

    # Any option whose value is a Docker context can be completed dynamically.
    case $prev in
        -c|--context)
            _fdoc_complete__contexts_reply "$cur"
            return 0
            ;;
    esac

    # Compose file/path options.
    case $prev in
        -f|--file)
            mapfile -t COMPREPLY < <(compgen -f -- "$cur")
            return 0
            ;;
        --project-directory)
            mapfile -t COMPREPLY < <(compgen -d -- "$cur")
            return 0
            ;;
        --profile|--env-file)
            # Profiles are project-dependent and env files are ordinary paths.
            if [[ $prev == --env-file ]]; then
                mapfile -t COMPREPLY < <(compgen -f -- "$cur")
            fi
            return 0
            ;;
    esac

    # Handle attached forms such as --context=foo and --file=compose.yaml.
    case $cur in
        --context=*|-c=*)
            local context_prefix context_value i
            if [[ $cur == --context=* ]]; then
                context_prefix='--context='
            else
                context_prefix='-c='
            fi
            context_value=${cur#*=}
            _fdoc_complete__contexts_reply "$context_value"
            for i in "${!COMPREPLY[@]}"; do
                COMPREPLY[i]="${context_prefix}${COMPREPLY[i]}"
            done
            return 0
            ;;
        --file=*|-f=*)
            local prefix value
            if [[ $cur == --file=* ]]; then
                prefix='--file='
            else
                prefix='-f='
            fi
            value=${cur#*=}
            local -a matches=()
            while IFS= read -r i; do
                [[ -n $i ]] || continue
                matches+=("${prefix}${i}")
            done < <(_fdoc_complete__files "$value")
            COMPREPLY=("${matches[@]}")
            return 0
            ;;
        --project-directory=*)
            local prefix_dir value_dir
            prefix_dir="${cur%%=*}="
            value_dir="${cur#*=}"
            local -a dir_matches=()
            while IFS= read -r i; do
                [[ -n $i ]] || continue
                dir_matches+=("${prefix_dir}${i}")
            done < <(_fdoc_complete__dirs "$value_dir")
            COMPREPLY=("${dir_matches[@]}")
            return 0
            ;;
    esac

    raw_resource=$(_fdoc_complete__find_resource words "$COMP_CWORD" 2>/dev/null || true)
    normalized_resource=$(_fdoc_complete__resource "$raw_resource" 2>/dev/null || true)

    if [[ -z $normalized_resource ]]; then
        # Resource/subcommand completion is intentionally explicit. This keeps
        # arbitrary Docker/fzf search text from turning into filename noise.
        local -a top_level=(
            'containers' 'container' 'ps'
            'images' 'image' 'img'
            'volumes' 'volume' 'vol'
            'networks' 'network' 'net'
            'compose' 'composes' 'services' 'service'
            'ctx' 'context' 'contexts'
            'resource' 'resources' 'type' 'types'
            'help'
        )

        if [[ $cur == -* ]]; then
            _fdoc_complete__values_reply "$cur" \
                '--context' '--help' '-h' '--version' '-V'
        else
            _fdoc_complete__values_reply "$cur" "${top_level[@]}"
        fi
        return 0
    fi

    # Global context/help options are accepted after engine resources as well.
    case $cur in
        --*|-*)
            case $normalized_resource in
                compose)
                    _fdoc_complete__values_reply "$cur" \
                        '-c' '--context' '-f' '--file' '-p' '--project-name' \
                        '--project-directory' '--profile' '--env-file' '--help' '-h'
                    ;;
                ctx)
                    _fdoc_complete__values_reply "$cur" \
                        '--help' '-h'
                    ;;
                resource)
                    _fdoc_complete__values_reply "$cur" '--help' '-h'
                    ;;
                *)
                    _fdoc_complete__values_reply "$cur" \
                        '-c' '--context' '--help' '-h'
                    ;;
            esac
            return 0
            ;;
    esac

    # Compose service names are intentionally left as free-form fzf queries.
    # Engine resources and Compose all accept arbitrary search terms here.
    return 0
}

# Complete both the canonical command and the user's planned alias. Using one
# completion function means an alias such as `alias ffd='fdoc'` gets identical
# behavior without requiring the alias to exist when this file is loaded.
complete -F _fdoc_complete fdoc ffd

# vim: set ft=bash:
