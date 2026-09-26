# --- kubectl -----------------------------------------------------------------
# Fish helpers for kubectl.
#
# Design goals:
#   - No kubectl/API calls while this file is sourced.
#   - Thin wrappers: arguments remain fully under the caller's control.
#   - No destructive defaults (`delete`, `apply`, `edit`, etc. stay explicit).
#   - Native kubectl completion remains authoritative for every wrapper.
#   - Fish's native `--wraps` mechanism handles wrapper completion.
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


# Remove previous definitions, equivalent to the Bash `unalias` block.
# Suppress errors because functions may not exist yet.
for __n in k kg kgp kgpa kgn kgno kgs kgd kgds kgst kgc kgsec kgj \
           kge kga kgwide kgyaml kgjson kw kp kwp kdesc kapply kaf \
           kdiff kdel kedit kpatch kcreate krun kx kattach kl kpf ktop \
           kevents kapi kapires kexplain kauth kcan kcfg kctx kctxs kusectx \
           kns ksetns kroll kset kscale klabel kann ktaint kcordon kuncordon \
           kdrain kresources
    functions -e $__n 2>/dev/null
end
set -e __n


# Do not execute any kubectl command while sourcing.
# `command -sq` only checks whether the executable can be found.
if command -sq kubectl

    # Default only when the variable is not already defined.
    set -q KUBE_LOG_TAIL; or set -gx KUBE_LOG_TAIL 1000


    ##### CORE

    function k --wraps kubectl
        command kubectl $argv
    end


    ##### GET / WATCH

    function kg --wraps 'kubectl get'
        command kubectl get $argv
    end

    function kgp --wraps 'kubectl get pods'
        command kubectl get pods $argv
    end

    function kgpa --wraps 'kubectl get pods'
        command kubectl get pods -A $argv
    end

    function kgn --wraps 'kubectl get nodes'
        command kubectl get nodes $argv
    end

    function kgno --wraps 'kubectl get nodes'
        command kubectl get nodes -o wide $argv
    end

    function kgs --wraps 'kubectl get svc'
        command kubectl get svc $argv
    end

    function kgd --wraps 'kubectl get deploy'
        command kubectl get deploy $argv
    end

    function kgds --wraps 'kubectl get daemonsets'
        command kubectl get daemonsets $argv
    end

    function kgst --wraps 'kubectl get statefulsets'
        command kubectl get statefulsets $argv
    end

    function kgc --wraps 'kubectl get configmaps'
        command kubectl get configmaps $argv
    end

    function kgsec --wraps 'kubectl get secrets'
        command kubectl get secrets $argv
    end

    function kgj --wraps 'kubectl get jobs'
        command kubectl get jobs $argv
    end

    function kge --wraps 'kubectl get events'
        command kubectl get events $argv
    end

    function kga --wraps 'kubectl get all'
        command kubectl get all $argv
    end

    function kgwide --wraps 'kubectl get'
        command kubectl get -o wide $argv
    end

    function kgyaml --wraps 'kubectl get'
        command kubectl get -o yaml $argv
    end

    function kgjson --wraps 'kubectl get'
        command kubectl get -o json $argv
    end

    function kw --wraps 'kubectl get'
        command kubectl get --watch $argv
    end

    function kp --wraps 'kubectl get pods'
        command kubectl get pods $argv
    end

    function kwp --wraps 'kubectl get pods'
        command kubectl get pods --watch $argv
    end


    ##### INSPECTION / CHANGE REVIEW

    function kdesc --wraps 'kubectl describe'
        command kubectl describe $argv
    end

    function kdiff --wraps 'kubectl diff'
        command kubectl diff $argv
    end

    function kexplain --wraps 'kubectl explain'
        command kubectl explain $argv
    end

    function kapi --wraps 'kubectl api-versions'
        command kubectl api-versions $argv
    end

    function kapires --wraps 'kubectl api-resources'
        command kubectl api-resources $argv
    end

    function kresources --wraps 'kubectl api-resources'
        command kubectl api-resources $argv
    end


    ##### MUTATION
    #
    # Intentionally explicit:
    # no implicit force / yes / prune.

    function kapply --wraps 'kubectl apply'
        command kubectl apply $argv
    end

    function kaf --wraps 'kubectl apply'
        command kubectl apply -f $argv
    end

    function kdel --wraps 'kubectl delete'
        command kubectl delete $argv
    end

    function kcreate --wraps 'kubectl create'
        command kubectl create $argv
    end

    function kedit --wraps 'kubectl edit'
        command kubectl edit $argv
    end

    function kpatch --wraps 'kubectl patch'
        command kubectl patch $argv
    end

    function kset --wraps 'kubectl set'
        command kubectl set $argv
    end

    function kscale --wraps 'kubectl scale'
        command kubectl scale $argv
    end

    function klabel --wraps 'kubectl label'
        command kubectl label $argv
    end

    function kann --wraps 'kubectl annotate'
        command kubectl annotate $argv
    end

    function ktaint --wraps 'kubectl taint'
        command kubectl taint $argv
    end

    function kcordon --wraps 'kubectl cordon'
        command kubectl cordon $argv
    end

    function kuncordon --wraps 'kubectl uncordon'
        command kubectl uncordon $argv
    end

    function kdrain --wraps 'kubectl drain'
        command kubectl drain $argv
    end

    function kroll --wraps 'kubectl rollout'
        command kubectl rollout $argv
    end


    ##### INTERACTIVE / RUNTIME

    function krun --wraps 'kubectl run'
        if isatty stdin; and isatty stdout
            command kubectl run $argv --rm -it
        else
            command kubectl run $argv --rm -i
        end
    end

    function kx --wraps 'kubectl exec'
        # kx <pod> -> best available shell
        # kx <args...> -> plain exec

        if isatty stdin; and isatty stdout
            set -l ti -it
        else
            set -l ti -i
        end

        if test (count $argv) -eq 1
            command kubectl exec $ti $argv[1] -- sh -c \
                'command -v bash >/dev/null 2>&1 && exec bash || exec sh'
        else
            command kubectl exec $ti $argv
        end
    end

    function kattach --wraps 'kubectl attach'
        if isatty stdin; and isatty stdout
            command kubectl attach -it $argv
        else
            command kubectl attach $argv
        end
    end

    function kl --wraps 'kubectl logs'
        command kubectl logs --tail $KUBE_LOG_TAIL -f $argv
    end

    function kpf --wraps 'kubectl port-forward'
        command kubectl port-forward $argv
    end

    function ktop --wraps 'kubectl top'
        command kubectl top $argv
    end

    function kevents --wraps 'kubectl get events'
        command kubectl get events --sort-by='.lastTimestamp' $argv
    end


    ##### AUTH / CONFIG

    function kauth --wraps 'kubectl auth'
        command kubectl auth $argv
    end

    function kcan --wraps 'kubectl auth can-i'
        command kubectl auth can-i $argv
    end

    function kcfg --wraps 'kubectl config'
        command kubectl config $argv
    end

    function kctx --wraps 'kubectl config current-context'
        command kubectl config current-context $argv
    end

    function kctxs --wraps 'kubectl config get-contexts'
        command kubectl config get-contexts $argv
    end

    function kusectx --wraps 'kubectl config use-context'
        command kubectl config use-context $argv
    end

    function kns --wraps 'kubectl get namespaces'
        command kubectl get namespaces $argv
    end

    function ksetns --wraps 'kubectl config set-context'
        if test (count $argv) -ne 1
            printf 'usage: ksetns NAMESPACE\n' >&2
            return 2
        end

        command kubectl config set-context --current --namespace=$argv[1]
    end


    ##### COMPLETION
    #
    # Fish does not need Bash's COMP_WORDS / COMP_CWORD / COMP_LINE /
    # COMP_POINT rewriting.
    #
    # `--wraps` tells fish which underlying command's completion to inherit.
    #
    # For wrappers that add fixed arguments, the function itself preserves
    # those arguments while completion is delegated to the corresponding
    # kubectl command.
    #
    # These declarations are intentionally kept in this same file.

    complete -c k          --wraps kubectl

    complete -c kg         --wraps 'kubectl get'
    complete -c kgp        --wraps 'kubectl get pods'
    complete -c kgpa       --wraps 'kubectl get pods'
    complete -c kgn        --wraps 'kubectl get nodes'
    complete -c kgno       --wraps 'kubectl get nodes'
    complete -c kgs        --wraps 'kubectl get svc'
    complete -c kgd        --wraps 'kubectl get deploy'
    complete -c kgds       --wraps 'kubectl get daemonsets'
    complete -c kgst       --wraps 'kubectl get statefulsets'
    complete -c kgc        --wraps 'kubectl get configmaps'
    complete -c kgsec      --wraps 'kubectl get secrets'
    complete -c kgj        --wraps 'kubectl get jobs'
    complete -c kge        --wraps 'kubectl get events'
    complete -c kga        --wraps 'kubectl get all'
    complete -c kgwide     --wraps 'kubectl get'
    complete -c kgyaml     --wraps 'kubectl get'
    complete -c kgjson     --wraps 'kubectl get'
    complete -c kw         --wraps 'kubectl get'
    complete -c kp         --wraps 'kubectl get pods'
    complete -c kwp        --wraps 'kubectl get pods'

    complete -c kdesc      --wraps 'kubectl describe'
    complete -c kdiff      --wraps 'kubectl diff'
    complete -c kexplain   --wraps 'kubectl explain'
    complete -c kapi       --wraps 'kubectl api-versions'
    complete -c kapires    --wraps 'kubectl api-resources'
    complete -c kresources --wraps 'kubectl api-resources'

    complete -c kapply     --wraps 'kubectl apply'
    complete -c kaf        --wraps 'kubectl apply'
    complete -c kdel       --wraps 'kubectl delete'
    complete -c kcreate    --wraps 'kubectl create'
    complete -c kedit      --wraps 'kubectl edit'
    complete -c kpatch     --wraps 'kubectl patch'
    complete -c kset       --wraps 'kubectl set'
    complete -c kscale     --wraps 'kubectl scale'
    complete -c klabel     --wraps 'kubectl label'
    complete -c kann       --wraps 'kubectl annotate'
    complete -c ktaint     --wraps 'kubectl taint'
    complete -c kcordon    --wraps 'kubectl cordon'
    complete -c kuncordon  --wraps 'kubectl uncordon'
    complete -c kdrain     --wraps 'kubectl drain'
    complete -c kroll      --wraps 'kubectl rollout'

    complete -c krun       --wraps 'kubectl run'
    complete -c kx         --wraps 'kubectl exec'
    complete -c kattach   --wraps 'kubectl attach'
    complete -c kl         --wraps 'kubectl logs'
    complete -c kpf        --wraps 'kubectl port-forward'
    complete -c ktop       --wraps 'kubectl top'
    complete -c kevents   --wraps 'kubectl get events'

    complete -c kauth      --wraps 'kubectl auth'
    complete -c kcan       --wraps 'kubectl auth can-i'
    complete -c kcfg       --wraps 'kubectl config'
    complete -c kctx       --wraps 'kubectl config current-context'
    complete -c kctxs      --wraps 'kubectl config get-contexts'
    complete -c kusectx    --wraps 'kubectl config use-context'
    complete -c kns        --wraps 'kubectl get namespaces'
    complete -c ksetns     --wraps 'kubectl config set-context'
end
# vim: set ft=fish:
