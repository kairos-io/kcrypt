VERSION 0.6
# Note the base image needs to have dracut.
# TODO: This needs to come from pre-built kernels in c3os repos, kcrypt included.
# Framework images should use our initrd
ARG BASE_IMAGE=quay.io/kairos/core-opensuse

build-kcrypt:
    FROM golang:alpine
    COPY . /work
    WORKDIR /work
    RUN CGO_ENABLED=0 go build -o kcrypt
    SAVE ARTIFACT /work/kcrypt AS LOCAL kcrypt

build-dracut:
    FROM $BASE_IMAGE
    COPY . /work
    COPY +build-kcrypt/kcrypt /usr/bin/kcrypt
    WORKDIR /work
    RUN cp -r dracut/* /usr/lib/dracut/modules.d
    RUN cp dracut.conf /etc/dracut.conf.d/10-kcrypt.conf
    RUN kernel=$(ls /lib/modules | head -n1) && \
        dracut -f "/boot/initrd-${kernel}" "${kernel}" && \
        ln -sf "initrd-${kernel}" /boot/initrd
    ARG INITRD=$(readlink -f /boot/initrd)
    SAVE ARTIFACT $INITRD AS LOCAL initrd

image:
    FROM $BASE_IMAGE
    ARG IMAGE=dracut
    ARG INITRD=$(readlink -f /boot/initrd)
    ARG NAME=$(basename $INITRD)
    COPY +build-dracut/$NAME $INITRD
    COPY +build-kcrypt/kcrypt /usr/bin/kcrypt
    # This is the discovery plugin that needs to be replaced!
    # TODO: After install, copy any discovery plugin found to /oem - this is weak - another way is to inject the binary into the initrd, but a the moment it is not working properly.
    COPY +dummy-discovery/dummy-discovery /system/discovery/kcrypt-discovery-dummy
    # XXX: this is not working properly, but avoids the /oem copy.
    #RUN kcrypt inject-initrd $INITRD /system/discovery/kcrypt-discovery-dummy /system/discovery/kcrypt-discovery-dummy
    SAVE IMAGE $IMAGE

dummy-discovery:
    FROM golang:alpine
    COPY . /work
    WORKDIR /work
    RUN CGO_ENABLED=0 go build -o dummy-discovery ./examples/dummy-discovery
    SAVE ARTIFACT /work/dummy-discovery AS LOCAL kcrypt-discovery-dummy


iso: 
    ARG ISO_NAME=test
    FROM quay.io/kairos/osbuilder-tools

    WORKDIR /build
    RUN zypper in -y jq docker wget
    RUN mkdir -p files-iso/boot/grub2
    RUN wget https://raw.githubusercontent.com/c3os-io/c3os/master/overlay/files-iso/boot/grub2/grub.cfg -O files-iso/boot/grub2/grub.cfg
    WITH DOCKER --allow-privileged --load $IMG=(+image)
        RUN /entrypoint.sh --name $ISO_NAME --debug build-iso --date=false --local --overlay-iso /build/files-iso dracut:latest --output /build/
    END
   # See: https://github.com/rancher/elemental-cli/issues/228
    RUN sha256sum $ISO_NAME.iso > $ISO_NAME.iso.sha256
    SAVE ARTIFACT /build/$ISO_NAME.iso iso AS LOCAL build/$ISO_NAME.iso
    SAVE ARTIFACT /build/$ISO_NAME.iso.sha256 sha256 AS LOCAL build/$ISO_NAME.iso.sha256
