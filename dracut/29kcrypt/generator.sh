#!/bin/bash

type getarg >/dev/null 2>&1 || . /lib/dracut-lib.sh

GENERATOR_DIR="$2"

[ -z "$GENERATOR_DIR" ] && exit 1
[ -d "$GENERATOR_DIR" ] || mkdir "$GENERATOR_DIR"

if getargbool 0 rd.neednet; then
    {
        echo "[Unit]"
        echo "DefaultDependencies=no"
        echo "Description=kcrypt online mount"
        echo "Before=cos-immutable-rootfs.service"
        echo "After=network-online.target"
        echo "Wants=network-online.target"
        echo "[Service]"
        echo "Type=oneshot"
        echo "RemainAfterExit=no"
        echo "ExecStart=/sbin/kcrypt-mount-local"
    } > "$GENERATOR_DIR"/kcrypt.service
else
    {
        echo "[Unit]"
        echo "DefaultDependencies=no"
        echo "Description=kcrypt mount"
        echo "Before=cos-immutable-rootfs.service"
        echo "[Service]"
        echo "Type=oneshot"
        echo "RemainAfterExit=no"
        echo "ExecStart=/sbin/kcrypt-mount-local"
    } > "$GENERATOR_DIR"/kcrypt.service
fi

if [ ! -e "$GENERATOR_DIR/initrd-fs.target.requires/kcrypt.service" ]; then
    mkdir -p "$GENERATOR_DIR"/initrd-fs.target.requires
    ln -s "$GENERATOR_DIR"/kcrypt.service \
        "$GENERATOR_DIR"/initrd-fs.target.requires/kcrypt.service
fi