#!/bin/sh
# -*- mode: shell-script; indent-tabs-mode: nil; sh-basic-offset: 4; -*-
# ex: ts=8 sw=4 sts=4 et filetype=sh

type getarg > /dev/null 2>&1 || . /lib/dracut-lib.sh

PATH=/usr/sbin:/usr/bin:/sbin:/bin

OEM=$(blkid -L COS_OEM)
if [ "$OEM" != "" ]; then
    mkdir /oem
    mount $OEM /oem
fi

kcrypt unlock-all

if [ "$OEM" != "" ]; then
umount /oem
fi
