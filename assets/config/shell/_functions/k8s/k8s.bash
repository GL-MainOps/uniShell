# --- kubectl -----------------------------------------------------------------
# Bash helpers for kubectl.  Native kubectl completion is delegated to the
# completion function already provided by kubectl/bash-completion.
#
# Design goals:
#   - No kubectl/API calls while this file is sourced.
#   - Thin wrappers: arguments remain fully under the caller's control.
#   - No destructive defaults (`delete`, `apply`, `edit`, etc. stay explicit).
#   - Native completion remains authoritative for every wrapper.
#   - Completion is resolved lazily, so startup stays cheap.
#   - Bash 4.2+ to match the associative-array implementation used here.
#
# Examples:
#   k                         # kubectl
#   kg pods                   # kubectl get pods
#   kgp                       # kubectl get pods
#   kgpa                      # kubectl get pods -A
#   kns                       # kubectl get namespaces
#   kctx                      # current context
#   kctxs                     # list contexts
#   kusectx <context>         # switch context
#   kl <pod>                  # follow pod logs, last KUBE_LOG_TAIL lines
#   kx <pod>                  # best available shell
#   kdel ...                  # kubectl delete (native behavior; no auto-force)
#
# Environment:
#   KUBE_LOG_TAIL             default log tail for `kl` (default: 1000)

for __n in k kg kgp kgpa kgn kgno kgs kgd kgds kgst kgc kgsec kgj \
           kge kga kgwide kgyaml kgjson kw kp kwp kdesc kapply kaf \
           kdiff kdel kedit kpatch kcreate krun kx kattach kl kpf ktop \
           kevents kapi kapires kexplain kauth kcan kcfg kctx kctxs kusectx \
           kns ksetns kroll kset kscale klabel kann ktaint kcordon kuncordon \
           kdrain kresources; do
    unalias "$__n" 2>/dev/null
done
unset __n

if [[ $- == *i* ]] && command -v kubectl &>/dev/null &&
   (( BASH_VERSINFO[0] > 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] >= 2) )); then

    : "${KUBE_LOG_TAIL:=1000}"

    ##### CORE
    k()       { kubectl "$@"; }

    ##### GET / WATCH
    kg()      { kubectl get "$@"; }
    kgp()     { kubectl get pods "$@"; }
    kgpa()    { kubectl get pods -A "$@"; }
    kgn()     { kubectl get nodes "$@"; }
    kgno()    { kubectl get nodes -o wide "$@"; }
    kgs()     { kubectl get svc "$@"; }
    kgd()     { kubectl get deploy "$@"; }
    kgds()    { kubectl get daemonsets "$@"; }
    kgst()    { kubectl get statefulsets "$@"; }
    kgc()     { kubectl get configmaps "$@"; }
    kgsec()   { kubectl get secrets "$@"; }
    kgj()     { kubectl get jobs "$@"; }
    kge()     { kubectl get events "$@"; }
    kga()     { kubectl get all "$@"; }
    kgwide()  { kubectl get -o wide "$@"; }
    kgyaml()  { kubectl get -o yaml "$@"; }
    kgjson()  { kubectl get -o json "$@"; }
    kw()      { kubectl get --watch "$@"; }
    kp()      { kubectl get pods "$@"; }
    kwp()     { kubectl get pods --watch "$@"; }

    ##### INSPECTION / CHANGE REVIEW
    kdesc()   { kubectl describe "$@"; }
    kdiff()   { kubectl diff "$@"; }
    kexplain(){ kubectl explain "$@"; }
    kapi()    { kubectl api-versions "$@"; }
    kapires() { kubectl api-resources "$@"; }
    kresources() { kubectl api-resources "$@"; }

    ##### MUTATION -- intentionally explicit, no implicit force/yes/prune
    kapply()  { kubectl apply "$@"; }
    kaf()     { kubectl apply -f "$@"; }
    kdel()    { kubectl delete "$@"; }
    kcreate() { kubectl create "$@"; }
    kedit()   { kubectl edit "$@"; }
    kpatch()  { kubectl patch "$@"; }
    kset()    { kubectl set "$@"; }
    kscale()  { kubectl scale "$@"; }
    klabel()  { kubectl label "$@"; }
    kann()    { kubectl annotate "$@"; }
    ktaint()  { kubectl taint "$@"; }
    kcordon() { kubectl cordon "$@"; }
    kuncordon(){ kubectl uncordon "$@"; }
    kdrain()  { kubectl drain "$@"; }
    kroll()   { kubectl rollout "$@"; }

    ##### INTERACTIVE / RUNTIME
    krun() {
        local -a ti=(-i)
        [[ -t 0 && -t 1 ]] && ti=(-it)
        kubectl run "$@" --rm "${ti[@]}"
    }

    kx() {  # kx <pod> -> best available shell; kx <args...> -> plain exec
        local ti=-i
        [[ -t 0 && -t 1 ]] && ti=-it
        if (( $# == 1 )); then
            kubectl exec "$ti" "$1" -- sh -c \
                'command -v bash >/dev/null 2>&1 && exec bash || exec sh'
        else
            kubectl exec "$ti" "$@"
        fi
    }

    kattach() {
        local -a ti=()
        [[ -t 0 && -t 1 ]] && ti=(-it)
        kubectl attach "${ti[@]}" "$@"
    }

    kl()      { kubectl logs --tail "$KUBE_LOG_TAIL" -f "$@"; }
    kpf()     { kubectl port-forward "$@"; }
    ktop()    { kubectl top "$@"; }
    kevents() { kubectl get events --sort-by='.lastTimestamp' "$@"; }

    ##### AUTH / CONFIG
    kauth()   { kubectl auth "$@"; }
    kcan()    { kubectl auth can-i "$@"; }
    kcfg()    { kubectl config "$@"; }
    kctx()    { kubectl config current-context "$@"; }
    kctxs()   { kubectl config get-contexts "$@"; }
    kusectx() { kubectl config use-context "$@"; }
    kns()     { kubectl get namespaces "$@"; }
    ksetns() {
        (( $# == 1 )) || {
            printf 'usage: ksetns NAMESPACE\n' >&2
            return 2
        }
        kubectl config set-context --current --namespace="$1"
    }

    ##### COMPLETION: wrapper -> kubectl command it completes as
    # The values are command/argument prefixes inserted before the wrapper's
    # original arguments.  Native kubectl completion then handles the rest.
    declare -gA KUBECTL_WRAPPER_MAP=(
        [k]=""

        [kg]="get"
        [kgp]="get pods"
        [kgpa]="get pods -A"
        [kgn]="get nodes"
        [kgno]="get nodes -o wide"
        [kgs]="get svc"
        [kgd]="get deploy"
        [kgds]="get daemonsets"
        [kgst]="get statefulsets"
        [kgc]="get configmaps"
        [kgsec]="get secrets"
        [kgj]="get jobs"
        [kge]="get events"
        [kga]="get all"
        [kgwide]="get -o wide"
        [kgyaml]="get -o yaml"
        [kgjson]="get -o json"
        [kw]="get --watch"
        [kp]="get pods"
        [kwp]="get pods --watch"

        [kdesc]="describe"
        [kdiff]="diff"
        [kexplain]="explain"
        [kapi]="api-versions"
        [kapires]="api-resources"
        [kresources]="api-resources"

        [kapply]="apply"
        [kaf]="apply -f"
        [kdel]="delete"
        [kcreate]="create"
        [kedit]="edit"
        [kpatch]="patch"
        [kset]="set"
        [kscale]="scale"
        [klabel]="label"
        [kann]="annotate"
        [ktaint]="taint"
        [kcordon]="cordon"
        [kuncordon]="uncordon"
        [kdrain]="drain"
        [kroll]="rollout"

        [krun]="run"
        [kx]="exec"
        [kattach]="attach"
        [kl]="logs"
        [kpf]="port-forward"
        [ktop]="top"
        [kevents]="get events --sort-by=.lastTimestamp"

        [kauth]="auth"
        [kcan]="auth can-i"
        [kcfg]="config"
        [kctx]="config current-context"
        [kctxs]="config get-contexts"
        [kusectx]="config use-context"
        [kns]="get namespaces"
        [ksetns]="config set-context --current --namespace"
    )

    __kubectl_wrapper_complete() {
        local wrapper=${COMP_WORDS[0]} IFS=$' \t\n'
        [[ -n ${KUBECTL_WRAPPER_MAP[$wrapper]+_} ]] || return 1

        # Prefer the already-sourced native completion.  Fall back to the
        # bash-completion loader, then kubectl itself, but only on demand.
        if ! declare -F __start_kubectl >/dev/null; then
            declare -F _completion_loader >/dev/null &&
                _completion_loader kubectl &>/dev/null
            declare -F __start_kubectl >/dev/null ||
                source <(kubectl completion bash 2>/dev/null) 2>/dev/null
            declare -F __start_kubectl >/dev/null || return 1
        fi

        local -a exp
        read -ra exp <<< "${KUBECTL_WRAPPER_MAP[$wrapper]}"

        # Rebuild the command line exactly as native kubectl completion expects.
        # Keep leading whitespace and the cursor position consistent.
        local prefix=kubectl
        (( ${#exp[@]} )) && prefix+=" ${exp[*]}"

        local lead=${COMP_LINE%%[![:space:]]*}
        local rest=${COMP_LINE#"$lead$wrapper"}

        COMP_WORDS=(kubectl "${exp[@]}" "${COMP_WORDS[@]:1}")
        COMP_CWORD=$(( COMP_CWORD + ${#exp[@]} ))
        COMP_POINT=$(( COMP_POINT + ${#prefix} - ${#wrapper} ))
        COMP_LINE="$lead$prefix$rest"

        __start_kubectl
    }

    complete -o default -F __kubectl_wrapper_complete "${!KUBECTL_WRAPPER_MAP[@]}"
fi

