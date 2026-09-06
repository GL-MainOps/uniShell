# uniShell Step 14 shell-profile acceptance fixture.

set -gx UNISHELL_PROFILE_TEST "fish-profile-active"

function fish_prompt
    printf '[uniShell-profile] %s@%s:%s> ' $USER (hostname -s) (prompt_pwd)
end

alias uprofile 'printf "fish-profile-active\n"'
