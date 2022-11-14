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
RUN go mod tidy
RUN go clean
RUN go build --ldflags '-linkmode external -extldflags=-static'
RUN GOOS=windows go build
WORKDIR /app/cordial/integrations/servicenow
RUN go mod tidy
RUN go clean
RUN go build --ldflags '-linkmode external -extldflags=-static'
WORKDIR /app/cordial/integrations/pagerduty
RUN go mod tidy
RUN go clean
RUN go build --ldflags '-linkmode external -extldflags=-static'

# special centos7 build environment for shared libs
FROM centos:7 AS build-libs
LABEL stage=cordial-build
RUN yum install -y gcc make
ADD https://go.dev/dl/go1.19.3.linux-amd64.tar.gz /tmp/
RUN tar -C /usr/local -xzf /tmp/go1.19.3.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin
COPY ./ /app/cordial
WORKDIR /app/cordial/libraries/libemail
RUN make

FROM alpine AS cordial-build
LABEL stage=cordial-build
RUN apk add zip
WORKDIR /app/cordial
COPY --from=build /app/cordial/VERSION /
COPY --from=build /app/cordial/tools/geneos/geneos /cordial/bin/
COPY --from=build /app/cordial/tools/geneos/geneos.exe /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow/servicenow /app/cordial/integrations/servicenow/ticket.sh /app/cordial/integrations/pagerduty/pagerduty /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow/servicenow.example.yaml /app/cordial/integrations/pagerduty/cmd/pagerduty.defaults.yaml /cordial/etc/geneos/
COPY --from=build-libs /app/cordial/libraries/libemail/libemail.so /cordial/lib/
RUN mv /cordial /cordial-$(cat /VERSION)
WORKDIR /
RUN tar czf /cordial-$(cat /VERSION).tar.gz cordial-$(cat /VERSION) && zip -q -r /cordial-$(cat /VERSION).zip cordial-$(cat /VERSION) && rm -r /cordial-$(cat /VERSION)
CMD [ "bash" ]

FROM debian AS cordial-run
RUN apt update && apt install -y fontconfig ca-certificates
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]
