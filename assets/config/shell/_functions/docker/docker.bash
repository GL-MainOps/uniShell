for __n in d dps dimg dl dstop drm dins dnet dvol dprune dstats drun dex \
           dc dup ddown dcl dcr dcp co cou cod cog cor cop; do
    unalias "$__n" 2>/dev/null
done
unset __n

if [[ $- == *i* ]] && command -v docker &>/dev/null &&
   (( BASH_VERSINFO[0] > 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] >= 2) )); then

    : "${DOCKER_PROFILE:=$HOSTNAME}"   # compose profile for co* helpers (override in env)
    : "${DOCKER_LOG_TAIL:=1000}"

    ##### DOCKER
    d()      { docker "$@"; }
    dps()    { docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Image}}\t{{.Ports}}' "$@"; }
    dimg()   { docker images "$@"; }
    dl()     { docker logs --tail "$DOCKER_LOG_TAIL" -f "$@"; }
    dstop()  { docker stop "$@"; }
    drm()    { docker rm "$@"; }
    dins()   { docker inspect "$@"; }
    dnet()   { docker network "$@"; }
    dvol()   { docker volume "$@"; }
    dprune() { docker system prune "$@"; }   # no -f: keeps the confirmation prompt
    dstats() { docker stats --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}' "$@"; }
    drun()   { local ti=-i; [[ -t 0 && -t 1 ]] && ti=-it; docker run --rm "$ti" "$@"; }
    dex() {  # dex <ctr> -> best available shell; dex <args...> -> plain exec
        local ti=-i; [[ -t 0 && -t 1 ]] && ti=-it
        if (( $# == 1 )); then
            docker exec "$ti" "$1" sh -c 'command -v bash >/dev/null 2>&1 && exec bash || exec sh'
        else
            docker exec "$ti" "$@"
        fi
    }

    ##### COMPOSE (GENERIC)
    dc()     { docker compose "$@"; }
    dup()    { docker compose up -d "$@"; }
    ddown()  { docker compose down "$@"; }
    dcl()    { docker compose logs --tail "$DOCKER_LOG_TAIL" -f "$@"; }
    dcr()    { docker compose restart "$@"; }
    dcp()    { docker compose pull "$@"; }

    ##### COMPOSE (HOMELAB)   # cog, not col: avoids shadowing col(1)
    co()     { docker compose --profile "$DOCKER_PROFILE" "$@"; }
    cou()    { docker compose --profile "$DOCKER_PROFILE" up -d "$@"; }
    cod()    { docker compose --profile "$DOCKER_PROFILE" down "$@"; }
    cog()    { docker compose --profile "$DOCKER_PROFILE" logs --tail "$DOCKER_LOG_TAIL" -f "$@"; }
    cor()    { docker compose --profile "$DOCKER_PROFILE" restart "$@"; }
    cop()    { docker compose --profile "$DOCKER_PROFILE" pull "$@"; }

    ##### COMPLETION: wrapper -> docker subcommand it completes as
    declare -gA DOCKER_WRAPPER_MAP=(
        [d]=""
        [dps]="ps"          [dimg]="images"     [dl]="logs"
        [dstop]="stop"      [drm]="rm"          [dins]="inspect"
        [dnet]="network"    [dvol]="volume"     [dprune]="system prune"
        [dstats]="stats"    [drun]="run"        [dex]="exec"

        [dc]="compose"      [dup]="compose up"  [ddown]="compose down"
        [dcl]="compose logs" [dcr]="compose restart" [dcp]="compose pull"

        [co]="compose --profile $DOCKER_PROFILE"
        [cou]="compose --profile $DOCKER_PROFILE up"
        [cod]="compose --profile $DOCKER_PROFILE down"
        [cog]="compose --profile $DOCKER_PROFILE logs"
        [cor]="compose --profile $DOCKER_PROFILE restart"
        [cop]="compose --profile $DOCKER_PROFILE pull"
    )

    __docker_wrapper_complete() {
        local wrapper=${COMP_WORDS[0]} IFS=$' \t\n'
        [[ -n ${DOCKER_WRAPPER_MAP[$wrapper]+_} ]] || return 1

        # Resolve docker's native completion lazily (zero startup cost)
        if ! declare -F __start_docker >/dev/null; then
            declare -F _completion_loader >/dev/null && _completion_loader docker &>/dev/null
            declare -F __start_docker >/dev/null || source <(docker completion bash 2>/dev/null) 2>/dev/null
            declare -F __start_docker >/dev/null || return 1
        fi

        local -a exp
        read -ra exp <<< "${DOCKER_WRAPPER_MAP[$wrapper]}"
        local prefix=docker
        (( ${#exp[@]} )) && prefix+=" ${exp[*]}"

        # Rewrite all four COMP_* vars consistently; bash-completion derives `cur`
        # from COMP_LINE/COMP_POINT, so no empty-word " " hack is needed.
        local lead=${COMP_LINE%%[![:space:]]*}
        local rest=${COMP_LINE#"$lead$wrapper"}
        COMP_WORDS=(docker "${exp[@]}" "${COMP_WORDS[@]:1}")
        COMP_CWORD=$(( COMP_CWORD + ${#exp[@]} ))
        COMP_POINT=$(( COMP_POINT + ${#prefix} - ${#wrapper} ))
        COMP_LINE="$lead$prefix$rest"

        __start_docker
    }

    complete -o default -F __docker_wrapper_complete "${!DOCKER_WRAPPER_MAP[@]}"
fi
# vim: set ft=bash:
