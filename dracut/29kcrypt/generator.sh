#!/bin/bash

type getarg >/dev/null 2>&1 || . /lib/dracut-lib.sh

GENERATOR_DIR="$2"

[ -z "$GENERATOR_DIR" ] && exit 1
[ -d "$GENERATOR_DIR" ] || mkdir "$GENERATOR_DIR"

oem_label=$(getarg rd.cos.oemlabel=)

## Several things indicate booting from a different media so we should not do anything
## rd.cos.disable is set on LIVECD and disables mounting of any type
if getargbool 0 rd.cos.disable; then
    exit 0
fi
## Netboot is set on...well, netboot obiously
if getargbool 0 netboot; then
    exit 0
fi


# See https://github.com/kairos-io/packages/blob/d12b12b043a71d8471454f7b4fc84c3181d2bf60/packages/system/dracut/immutable-rootfs/30cos-immutable-rootfs/cos-generator.sh#L29
{
    echo "[Unit]"
    echo "DefaultDependencies=no"
    echo "Before=immucore.service"
    echo "Conflicts=initrd-switch-root.target"
    if getargbool 0 rd.neednet; then
        echo "Wants=network-online.target"
        echo "After=network-online.target"
        echo "Description=kcrypt online mount"
    else
        echo "Description=kcrypt mount"
    fi
    # OEM is special as kcrypt plugins might need that in order to unlock other partitions and plugins can reside in /oem as well and kcrypt needs to find them
    if [ -n "${oem_label}" ]; then
        echo "After=oem.mount"
    fi
    echo "After=sysroot.mount"
    echo "[Service]"
    echo "Type=oneshot"
    echo "RemainAfterExit=no"
    echo "ExecStart=/usr/bin/kcrypt unlock-all"
} > "$GENERATOR_DIR"/kcrypt.service


if [ ! -e "$GENERATOR_DIR/initrd-fs.target.requires/kcrypt.service" ]; then
    mkdir -p "$GENERATOR_DIR"/initrd-fs.target.requires
    ln -s "$GENERATOR_DIR"/kcrypt.service \
        "$GENERATOR_DIR"/initrd-fs.target.requires/kcrypt.service
fi
