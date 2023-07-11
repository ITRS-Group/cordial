# Dockerfile to build cordial components and tar.gz files
#
# Build executables statically on alpine for maximum compatibility, but
# then buils libemail.so (and future shared libraries) on a centos7
# image for glibc compatibility with Gateway on older systems.
#

ARG GOVERSION=1.20.5

# build Linux executables statically. Also build any Windows binaries
# here for completeness.
FROM golang:alpine AS build
LABEL stage=cordial-build

COPY go.mod go.sum cordial.go VERSION /app/cordial/
COPY integrations /app/cordial/integrations/
COPY pkg /app/cordial/pkg
COPY tools /app/cordial/tools

RUN set -eux; \
    apk add build-base; \
    cd /app/cordial/tools/geneos; \
    go build --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build --ldflags '-s -w'; \
    cd /app/cordial/tools/dv2email; \
    go build --ldflags '-s -w -linkmode external -extldflags=-static'; \
    cd /app/cordial/integrations/servicenow; \
    go build --ldflags '-s -w -linkmode external -extldflags=-static'; \
    cd /app/cordial/integrations/pagerduty; \
    go build --ldflags '-s -w -linkmode external -extldflags=-static'

# special centos7 build environment for shared libs and a version of the
# geneos program that is dynamic enough to communicate with a domain
# controller
FROM centos:7 AS build-libs
LABEL stage=cordial-build
RUN yum install -y gcc make
ARG GOVERSION
ARG BUILDOS
ARG BUILDARCH
ADD https://go.dev/dl/go${GOVERSION}.${BUILDOS}-${BUILDARCH}.tar.gz /tmp/
RUN tar -C /usr/local -xzf /tmp/go${GOVERSION}.${BUILDOS}-${BUILDARCH}.tar.gz
ENV PATH=$PATH:/usr/local/go/bin
COPY go.mod go.sum cordial.go VERSION /app/cordial/
COPY libraries /app/cordial/libraries/
COPY pkg /app/cordial/pkg
COPY tools /app/cordial/tools
RUN set -eux; \
    cd /app/cordial/tools/geneos; \
    go build --ldflags '-s -w'; \
    cd /app/cordial/libraries/libemail; \
    make; \
    cd /app/cordial/libraries/libalert; \
    make

#
# Build PDF documentation using mdpdf. Like all Puppeteer based PDF
# writers the support for relative links to files is broken, so
# documents with links to other docs in the same repo will be wrong.
#
FROM node AS build-docs
LABEL stage=cordial-build
RUN set -eux; \
    apt update; \
    apt install -y \
        libnss3 \
        libnspr4 \
        libatk1.0-0 \
        libatk-bridge2.0-0 \
        libcups2 \
        libdrm2 \
        libxkbcommon0 \
        libxcomposite1 \
        libxdamage1 \
        libxfixes3 \
        libxrandr2 \
        libgbm1 \
        libasound2; \
    npm install --global mdpdf; \
    npm install --global @mermaid-js/mermaid-cli

COPY go.mod go.sum cordial.go VERSION /app/cordial/
COPY integrations /app/cordial/integrations/
COPY libraries /app/cordial/libraries/
COPY pkg /app/cordial/pkg
COPY tools /app/cordial/tools

WORKDIR /app/cordial/doc-output
COPY tools/geneos/README.md geneos.md
COPY tools/dv2email/README.md dv2email.md
COPY integrations/servicenow/README.md servicenow.md
COPY integrations/pagerduty/README.md pagerduty.md
COPY libraries/libemail/README.md libemail.md
COPY libraries/libalert/README.md libalert.md

ARG MERMAID=".mermaid"
ARG READMEDIRS="tools/geneos tools/dv2email integrations/servicenow integrations/pagerduty libraries/libemail libraries/libalert"
RUN set -eux; \
    echo '{  "args": ["--no-sandbox"] }' > /puppeteer.json; \
    for i in ${READMEDIRS}; \
    do \
            mmdc -p /puppeteer.json -i /app/cordial/$i/README.md -o /app/cordial/$i/README${MERMAID}.md; \
            mdpdf --border=15mm /app/cordial/$i/README${MERMAID}.md ${i##*/}.pdf; \
    done

#
# assemble files from previous stages into a .tar.gz ready for extraction in the
# Makefile
#
FROM alpine AS cordial-build
LABEL stage=cordial-build
# WORKDIR /app/cordial
COPY --from=build /app/cordial/VERSION /
COPY --from=build /app/cordial/tools/geneos/geneos /cordial/bin/
COPY --from=build /app/cordial/tools/geneos/geneos.exe /cordial/bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /cordial/bin/
COPY --from=build-docs /app/cordial/doc-output /cordial/docs
COPY --from=build /app/cordial/integrations/servicenow/servicenow /app/cordial/integrations/pagerduty/pagerduty /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow/servicenow.example.yaml /app/cordial/integrations/pagerduty/cmd/pagerduty.defaults.yaml /cordial/etc/geneos/
COPY --from=build-libs /app/cordial/tools/geneos/geneos /cordial/bin/geneos.centos7-x86_64
COPY --from=build-libs /app/cordial/libraries/libemail/libemail.so /cordial/lib/
COPY --from=build-libs /app/cordial/libraries/libalert/libalert.so /cordial/lib/

RUN set -eux; \
        apk add upx; \
        upx -q /cordial/bin/*; \
        mv /cordial /cordial-$(cat /VERSION); \
        tar czf /cordial-$(cat /VERSION).tar.gz cordial-$(cat /VERSION)

CMD [ "bash" ]

#
# create a runnable test image
#
FROM debian AS cordial-run
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build-libs /app/cordial/libraries/libemail/libemail.so /lib/
RUN set -eux; \
    apt update; \
    apt install -y \
        fontconfig \
        ca-certificates \
        ; \
    useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos

CMD [ "bash" ]

#
# Without fontconfig the webserver will not start, but adding repos to
# centos8 is too hard for basic testing.
#
FROM centos:centos8 AS cordial-run-el8
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build-libs /app/cordial/libraries/libemail/libemail.so /lib/
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]
