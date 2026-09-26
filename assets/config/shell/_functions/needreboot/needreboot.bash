#!/usr/bin/env bash

needreboot() {
    local email="Muhammad Yahia"
    local distro=""

    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        distro="$ID"
    else
        echo "ERROR: Cannot determine OS (missing /etc/os-release). Unsupported system." >&2
        return 2
    fi

    local reboot_needed=0
    # local detail=""

    case "$distro" in
        ubuntu|debian)
            if [[ -f /var/run/reboot-required ]]; then
                reboot_needed=1
                # detail="$(cat /var/run/reboot-required.pkgs 2>/dev/null || cat /var/run/reboot-required)"
            fi
            ;;
        rhel|centos|fedora|rocky|almalinux)
            if command -v needs-restarting >/dev/null 2>&1; then
                if ! needs-restarting -r >/dev/null 2>&1; then
                    reboot_needed=1
                    # detail="$(needs-restarting -r 2>&1)"
                fi
            else
                echo "WARNING: 'needs-restarting' not found (install yum-utils/dnf-utils) to check reboot status." >&2
                return 2
            fi
            ;;
        arch|manjaro|endeavouros)
            local running_kernel installed_kernel
            running_kernel="$(uname -r)"
            if [[ -d /usr/lib/modules ]]; then
                installed_kernel="$(ls -1 /usr/lib/modules | sort -V | tail -n1)"
                if [[ -n "$installed_kernel" && "$installed_kernel" != "$running_kernel" ]]; then
                    reboot_needed=1
                    # detail="Running kernel: $running_kernel | Installed kernel: $installed_kernel"
                fi
            else
                echo "WARNING: Cannot locate /usr/lib/modules to compare kernel versions." >&2
                return 2
            fi
            ;;
        *)
            echo "ERROR: Unsupported distribution '$distro'. Cannot check reboot status." >&2
            return 2
            ;;
    esac

    if [[ "$reboot_needed" -eq 1 ]]; then
        echo -e "\n\nWARNING:\n\tSystem Needs to be Rebooted.\n\tKindly report this to \"$email\" as soon as you see this message!\n\n"
        # [[ -n "$detail" ]] && echo "$detail"
        return 0
    fi

    return 1
}
# vim: set ft=bash:
