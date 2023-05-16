# Dockerfile to build cordial components and tar.gz/zip files
#
# Build executables statically on alpine for maximum compatibility, but
# then buils libemail.so (and future shared libraries) on a centos7
# image for glibc compatibility with Gateway on older systems.
#

# build Linux executables statically. Also build any Windows binaries
# here for completeness.
FROM golang:alpine AS build
LABEL stage=cordial-build
# build-base required for support to build libemail (and CGO in the future)
RUN apk add build-base
# The "clean" lines below are in case of running this in a working
# directory with existing builds from outside the container, which may
# be from a different arch or environment
COPY ./ /app/cordial
WORKDIR /app/cordial/tools/geneos
RUN go build --ldflags '-linkmode external -extldflags=-static'
RUN GOOS=windows go build
WORKDIR /app/cordial/tools/dv2email
RUN go build --ldflags '-linkmode external -extldflags=-static'
WORKDIR /app/cordial/integrations/servicenow
RUN go build --ldflags '-linkmode external -extldflags=-static'
WORKDIR /app/cordial/integrations/pagerduty
RUN go build --ldflags '-linkmode external -extldflags=-static'

# special centos7 build environment for shared libs and a version of the
# geneos program that is dynamic enough to communicate with a domain
# controller
FROM centos:7 AS build-libs
LABEL stage=cordial-build
RUN yum install -y gcc make
ARG BUILDOS
ARG BUILDARCH
ADD https://go.dev/dl/go1.19.3.${BUILDOS}-${BUILDARCH}.tar.gz /tmp/
RUN tar -C /usr/local -xzf /tmp/go1.19.3.${BUILDOS}-${BUILDARCH}.tar.gz
ENV PATH=$PATH:/usr/local/go/bin
COPY ./ /app/cordial
WORKDIR /app/cordial/tools/geneos
RUN go build
WORKDIR /app/cordial/libraries/libemail
RUN make
WORKDIR /app/cordial/libraries/libalert
RUN make

#
# Build PDF documentation using mdpdf. Like all Puppeteer based PDF
# writers the support for relative links to files is broken, so
# documents with links to other docs in the same repo will be wrong.
#
FROM node AS build-docs
LABEL stage=cordial-build
COPY ./ /app/cordial
WORKDIR /app/cordial/doc-output
RUN apt update && apt install -y libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 libcups2 libdrm2 libxkbcommon0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 libasound2
RUN npm install --global mdpdf
RUN mdpdf --border=15mm /app/cordial/tools/geneos/README.md geneos.pdf
COPY ./tools/geneos/README.md geneos.md
RUN mdpdf --border=15mm /app/cordial/tools/dv2email/README.md dv2email.pdf
COPY ./tools/dv2email/README.md dv2email.md
RUN mdpdf --border=15mm /app/cordial/integrations/servicenow/README.md servicenow.pdf
COPY ./integrations/servicenow/README.md servicenow.md
RUN mdpdf --border=15mm /app/cordial/integrations/pagerduty/README.md pagerduty.pdf
COPY ./integrations/pagerduty/README.md pagerduty.md
RUN mdpdf --border=15mm /app/cordial/libraries/libemail/README.md libemail.pdf
COPY ./libraries/libemail/README.md libemail.md
RUN mdpdf --border=15mm /app/cordial/libraries/libalert/README.md libalert.pdf
COPY ./libraries/libalert/README.md libalert.md

#
# assemble files from previous stages into a .zip and .tar.gz ready from
# extraction in the Makefile
#
FROM alpine AS cordial-build
LABEL stage=cordial-build
RUN apk add zip
WORKDIR /app/cordial
COPY --from=build /app/cordial/VERSION /
COPY --from=build /app/cordial/tools/geneos/geneos /cordial/bin/
COPY --from=build /app/cordial/tools/geneos/geneos.exe /cordial/bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /cordial/bin/
COPY --from=build-docs /app/cordial/doc-output /cordial/docs
COPY --from=build /app/cordial/integrations/servicenow/servicenow /app/cordial/integrations/servicenow/ticket.sh /app/cordial/integrations/pagerduty/pagerduty /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow/servicenow.example.yaml /app/cordial/integrations/pagerduty/cmd/pagerduty.defaults.yaml /cordial/etc/geneos/
COPY --from=build-libs /app/cordial/tools/geneos/geneos /cordial/bin/geneos.centos7-x86_64
COPY --from=build-libs /app/cordial/libraries/libemail/libemail.so /cordial/lib/
COPY --from=build-libs /app/cordial/libraries/libalert/libalert.so /cordial/lib/
RUN mv /cordial /cordial-$(cat /VERSION)
WORKDIR /
RUN tar czf /cordial-$(cat /VERSION).tar.gz cordial-$(cat /VERSION) && zip -q -r /cordial-$(cat /VERSION).zip cordial-$(cat /VERSION)
CMD [ "bash" ]

#
# create a runnable test image
#
FROM debian AS cordial-run
RUN apt update && apt install -y fontconfig ca-certificates
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build-libs /app/cordial/libraries/libemail/libemail.so /lib/
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

#
# Without fontconfig the webserver will not start, but adding repos to
# centos8 is too hard for basic testing.
#
FROM centos:centos8 AS cordial-run-el8
# RUN apt update && apt install -y fontconfig ca-certificates
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build-libs /app/cordial/libraries/libemail/libemail.so /lib/
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]
