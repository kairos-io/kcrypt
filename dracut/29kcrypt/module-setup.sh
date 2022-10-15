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
    inst_simple "${moddir}/kcrypt.service" \
        "${systemdsystemunitdir}/kcrypt.service"

    inst_simple "${moddir}/kcrypt-online.service" \
        "${systemdsystemunitdir}/kcrypt-online.service"

    mkdir -p "${initdir}/${systemdsystemunitdir}/initrd-fs.target.requires"
    ln_r "../kcrypt.service" \
        "${systemdsystemunitdir}/initrd-fs.target.requires/kcrypt.service"
    ln_r "../kcrypt-online.service" \
        "${systemdsystemunitdir}/initrd-fs.target.requires/kcrypt-online.service"
    dracut_need_initqueue
}