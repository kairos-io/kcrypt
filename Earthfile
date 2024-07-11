VERSION 0.6
# Note the base image needs to have dracut.
# TODO: This needs to come from pre-built kernels in c3os repos, kcrypt included.
# Framework images should use our initrd
ARG BASE_IMAGE=quay.io/kairos/core-opensuse
# renovate: datasource=docker depName=golang
ARG GO_VERSION=1.22
# renovate: datasource=docker depName=golangci-lint
ARG GOLINT_VERSION=1.52.2

build-kcrypt:
    ARG GO_VERSION
    FROM golang:$GO_VERSION-alpine
    RUN apk add git
    COPY . /work
    WORKDIR /work
    ARG VERSION="$(git describe --tags)"
    RUN CGO_ENABLED=0 go build -o kcrypt -ldflags "-X main.Version=$VERSION"
    SAVE ARTIFACT /work/kcrypt kcrypt AS LOCAL kcrypt

dracut-artifacts:
    FROM $BASE_IMAGE
    WORKDIR /build
    COPY --dir dracut/29kcrypt .
    COPY dracut/10-kcrypt.conf .
    SAVE ARTIFACT 29kcrypt 29kcrypt
    SAVE ARTIFACT 10-kcrypt.conf 10-kcrypt.conf

build-dracut:
    FROM $BASE_IMAGE
    WORKDIR /work
    COPY +build-kcrypt/kcrypt /usr/bin/kcrypt
    COPY +dracut-artifacts/29kcrypt /usr/lib/dracut/modules.d/29kcrypt
    COPY +dracut-artifacts/10-kcrypt.conf /etc/dracut.conf.d/10-kcrypt.conf
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

lint:
    BUILD +golint
    BUILD +yamllint

golint:
    ARG GO_VERSION
    FROM golang:$GO_VERSION
    ARG GOLINT_VERSION
    RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v$GOLINT_VERSION
    WORKDIR /build
    COPY . .
    RUN golangci-lint run

yamllint:
    FROM cytopia/yamllint
    COPY . .
    RUN yamllint .github/workflows/
