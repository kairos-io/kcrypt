#!/bin/bash

# called by dracut
check() {
    require_binaries "$systemdutildir"/systemd || return 1
    return 255
}

# called by dracut 
depends() {
    echo systemd rootfs-block dm fs-lib 
    #tpm2-tss
    return 0
}

# called by dracut
installkernel() {
    instmods overlay
}

# called by dracut
install() {
    declare moddir=${moddir}
    declare systemdutildir=${systemdutildir}
    declare systemdsystemunitdir=${systemdsystemunitdir}
    declare initdir="${initdir}"

    inst_multiple \
        kcrypt
    inst_script "${moddir}/mount-local.sh" "/sbin/kcrypt-mount-local"
    #inst_hook pre-trigger 10 "$moddir/mount-local.sh"
    inst_script "${moddir}/generator.sh" \
        "${systemdutildir}/system-generators/dracut-kcrypt-generator"

    dracut_need_initqueue
}