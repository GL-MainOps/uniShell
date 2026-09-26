for __n in d dps dimg dl dstop drm dins dnet dvol dprune dstats drun dex \
            dc dup ddown dcl dcr dcp co cou cod cog cor cop
    functions -e $__n 2>/dev/null
end
set -e __n

if command -sq docker

    # compose profile for co* helpers (override in environment)
    set -q DOCKER_PROFILE; or set -gx DOCKER_PROFILE $hostname

    # Number of log lines to show
    set -q DOCKER_LOG_TAIL; or set -gx DOCKER_LOG_TAIL 1000


    ##### DOCKER

    function d --wraps docker
        command docker $argv
    end

    function dps --wraps 'docker ps'
        command docker ps \
            --format 'table {{.Names}}\t{{.Status}}\t{{.Image}}\t{{.Ports}}' \
            $argv
    end

    function dimg --wraps 'docker images'
        command docker images $argv
    end

    function dl --wraps 'docker logs'
        command docker logs --tail $DOCKER_LOG_TAIL -f $argv
    end

    function dstop --wraps 'docker stop'
        command docker stop $argv
    end

    function drm --wraps 'docker rm'
        command docker rm $argv
    end

    function dins --wraps 'docker inspect'
        command docker inspect $argv
    end

    function dnet --wraps 'docker network'
        command docker network $argv
    end

    function dvol --wraps 'docker volume'
        command docker volume $argv
    end

    function dprune --wraps 'docker system prune'
        command docker system prune $argv
    end

    function dstats --wraps 'docker stats'
        command docker stats \
            --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}' \
            $argv
    end

    function drun --wraps 'docker run'
        if isatty stdin; and isatty stdout
            command docker run --rm -it $argv
        else
            command docker run --rm -i $argv
        end
    end

    function dex --wraps 'docker exec'
        if isatty stdin; and isatty stdout
            set -l ti -it
        else
            set -l ti -i
        end

        if test (count $argv) -eq 1
            command docker exec $ti $argv[1] sh -c \
                'command -v bash >/dev/null 2>&1 && exec bash || exec sh'
        else
            command docker exec $ti $argv
        end
    end


    ##### COMPOSE (GENERIC)

    function dc --wraps 'docker compose'
        command docker compose $argv
    end

    function dup --wraps 'docker compose up'
        command docker compose up -d $argv
    end

    function ddown --wraps 'docker compose down'
        command docker compose down $argv
    end

    function dcl --wraps 'docker compose logs'
        command docker compose logs --tail $DOCKER_LOG_TAIL -f $argv
    end

    function dcr --wraps 'docker compose restart'
        command docker compose restart $argv
    end

    function dcp --wraps 'docker compose pull'
        command docker compose pull $argv
    end


    ##### COMPOSE (HOMELAB)   # cog, not col: avoids shadowing col(1)

    function co --wraps 'docker compose'
        command docker compose --profile $DOCKER_PROFILE $argv
    end

    function cou --wraps 'docker compose up'
        command docker compose --profile $DOCKER_PROFILE up -d $argv
    end

    function cod --wraps 'docker compose down'
        command docker compose --profile $DOCKER_PROFILE down $argv
    end

    function cog --wraps 'docker compose logs'
        command docker compose --profile $DOCKER_PROFILE \
            logs --tail $DOCKER_LOG_TAIL -f $argv
    end

    function cor --wraps 'docker compose restart'
        command docker compose --profile $DOCKER_PROFILE restart $argv
    end

    function cop --wraps 'docker compose pull'
        command docker compose --profile $DOCKER_PROFILE pull $argv
    end


    ##### COMPLETION
    #
    # Fish's `--wraps` replaces the Bash COMP_* rewriting machinery.
    # Docker's native fish completion is inherited by these wrappers.

    complete -c d     --wraps docker
    complete -c dps   --wraps 'docker ps'
    complete -c dimg  --wraps 'docker images'
    complete -c dl    --wraps 'docker logs'
    complete -c dstop --wraps 'docker stop'
    complete -c drm   --wraps 'docker rm'
    complete -c dins  --wraps 'docker inspect'
    complete -c dnet  --wraps 'docker network'
    complete -c dvol  --wraps 'docker volume'
    complete -c dprune --wraps 'docker system prune'
    complete -c dstats --wraps 'docker stats'
    complete -c drun  --wraps 'docker run'
    complete -c dex   --wraps 'docker exec'

    complete -c dc    --wraps 'docker compose'
    complete -c dup   --wraps 'docker compose up'
    complete -c ddown --wraps 'docker compose down'
    complete -c dcl   --wraps 'docker compose logs'
    complete -c dcr   --wraps 'docker compose restart'
    complete -c dcp   --wraps 'docker compose pull'

    complete -c co  --wraps 'docker compose'
    complete -c cou --wraps 'docker compose up'
    complete -c cod --wraps 'docker compose down'
    complete -c cog --wraps 'docker compose logs'
    complete -c cor --wraps 'docker compose restart'
    complete -c cop --wraps 'docker compose pull'

end
# vim: set ft=fish:
