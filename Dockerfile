# Dockerfile to build cordial components and tar.gz files

ARG GOVERSION=1.22.4

# The bullseye image seems to offer the most compatibility, including
# libemail.so dependencies
#
# Note: To build an executable for a modern Mac use something like:
#
# GOOS=darwin GOARCH=arm64 go build -o geneos.${GOOS}-${GOARCH} -tags netgo,osusergo --ldflags '-s -w'
# 
FROM golang:${GOVERSION}-bullseye AS build
LABEL stage=cordial-build
# base files
COPY go.mod go.sum cordial.go logging.go VERSION README.md CHANGELOG.md /app/cordial/
COPY pkg /app/cordial/pkg
# geneos, dv2email, gateway-reporter
COPY tools /app/cordial/tools
# servicenow, pagerduty
COPY integrations /app/cordial/integrations/
# libemail, libalerts
COPY libraries /app/cordial/libraries/
# gdna
COPY gdna /app/cordial/gdna
RUN --mount=type=cache,target=/root/.cache/go-build \
    set -eux; \
    # build geneos
    cd /app/cordial/tools/geneos; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build --ldflags '-s -w'; \
    # gateway-reporter
    cd /app/cordial/tools/gateway-reporter; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # dv2email
    cd /app/cordial/tools/dv2email; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # servicenow
    cd /app/cordial/integrations/servicenow; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # pagerduty
    cd /app/cordial/integrations/pagerduty; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # libemail
    cd /app/cordial/libraries/libemail; \
    make; \
    # libalerts
    cd /app/cordial/libraries/libalert; \
    make; \
    # gdna
    cd /app/cordial/gdna; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build --ldflags '-s -w';

#
# Build PDF documentation using mdpdf. Like all Puppeteer based PDF
# writers the support for relative links to files is broken, so
# documents with links to other docs in the same repo will be wrong.
#
FROM node AS cordial-docs
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    set -eux; \
    apt update; \
    apt install -y --no-install-recommends \
        fonts-noto-color-emoji \
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
    rm -fr /var/lib/apt/lists/*; \
    npm install --global mdpdf; \
    npm install --global @mermaid-js/mermaid-cli

COPY --from=build /app/cordial /app/cordial

WORKDIR /app/cordial/doc-output
COPY tools/geneos/README.md geneos.md
COPY tools/gateway-reporter/README.md gateway-reporter.md
COPY tools/dv2email/README.md dv2email.md
COPY tools/dv2email/screenshots/ screenshots/
COPY gdna/README.md gdna.md
COPY gdna/screenshots/ screenshots/
COPY integrations/servicenow/README.md servicenow.md
COPY integrations/pagerduty/README.md pagerduty.md
COPY libraries/libemail/README.md libemail.md
COPY libraries/libalert/README.md libalert.md

ARG MERMAID=".mermaid"
ARG READMEDIRS="tools/geneos tools/gateway-reporter tools/dv2email gdna integrations/servicenow integrations/pagerduty libraries/libemail libraries/libalert"
RUN set -eux; \
    echo '{  "args": ["--no-sandbox"] }' > /puppeteer.json; \
    for i in ${READMEDIRS}; \
    do \
            mmdc -p /puppeteer.json -i /app/cordial/$i/README.md -o /app/cordial/$i/README${MERMAID}.md; \
            mdpdf /app/cordial/$i/README${MERMAID}.md ${i##*/}.pdf --border=15mm --gh-style; \
    done; \
    mmdc -p /puppeteer.json -i /app/cordial/CHANGELOG.md -o /app/cordial/CHANGELOG${MERMAID}.md; \
    mdpdf /app/cordial/CHANGELOG${MERMAID}.md CHANGELOG.pdf --border=15mm --gh-style; \
    mmdc -p /puppeteer.json -i /app/cordial/README.md -o /app/cordial/README${MERMAID}.md; \
    mdpdf /app/cordial/README${MERMAID}.md README.pdf --border=15mm --gh-style

#
# assemble files from previous stages into a .tar.gz ready for extraction in the
# Makefile
#
FROM alpine AS cordial-build
LABEL stage=cordial-build

COPY --from=build /app/cordial/VERSION /
COPY --from=build /app/cordial/CHANGELOG.md /cordial/docs/
COPY --from=build /app/cordial/README.md /cordial/docs/

# tools binaries
COPY --from=build /app/cordial/tools/geneos/geneos /cordial/bin/
COPY --from=build /app/cordial/tools/geneos/geneos.exe /cordial/bin/
COPY --from=build /app/cordial/tools/gateway-reporter/gateway-reporter /cordial/bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /cordial/bin/

# docs
COPY --from=cordial-docs /app/cordial/doc-output /cordial/docs/

# servicenow
COPY --from=build /app/cordial/integrations/servicenow/servicenow /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow/servicenow.example.yaml /cordial/etc/geneos/

# pagerduty
COPY --from=build /app/cordial/integrations/pagerduty/pagerduty /cordial/bin/
COPY --from=build /app/cordial/integrations/pagerduty/cmd/pagerduty.defaults.yaml /cordial/etc/geneos/

# libemail / libalerts
COPY --from=build /app/cordial/libraries/libemail/libemail.so /cordial/lib/
COPY --from=build /app/cordial/libraries/libalert/libalert.so /cordial/lib/

# gdna
COPY --from=build /app/cordial/gdna/gdna /cordial/bin/
COPY --from=build /app/cordial/gdna/gdna.exe /cordial/bin/
COPY --from=build /app/cordial/gdna/cmd/gdna.defaults.yaml /cordial/etc/geneos/
COPY --from=build /app/cordial/gdna/gdna.yaml /cordial/etc/geneos/gdna.yaml
COPY --from=build /app/cordial/gdna/geneos/* /cordial/etc/geneos/gdna/

# build tar
RUN set -eux; \
    apk add upx tar; \
    upx -q /cordial/bin/*; \
    cp -r /cordial /cordial-$(cat /VERSION); \
    tar --sort=name -czf /cordial-$(cat /VERSION).tar.gz cordial-$(cat /VERSION)

CMD [ "bash" ]

# hand testing images

# create a runnable test image using basic debian
FROM debian AS cordial-run-debian
COPY --from=cordial-build /cordial/bin/geneos /bin/
COPY --from=cordial-build /cordial/bin/gateway-reporter /bin/
COPY --from=cordial-build /cordial/bin/dv2email /bin/
COPY --from=cordial-build /cordial/lib/libemail.so /lib/
RUN --mount=type=cache,target=/var/cache/apt \
    set -eux; \
    apt update; \
    apt install -y --no-install-recommends \
        fontconfig \
        ca-certificates \
        ; \
    useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# build a UBI8 for testing
FROM redhat/ubi8 AS cordial-run-ubi8
COPY --from=cordial-build /cordial/bin/geneos /bin/
COPY --from=cordial-build /cordial/bin/gdna /bin/
COPY --from=cordial-build /cordial/bin/gateway-reporter /bin/
COPY --from=cordial-build /cordial/bin/dv2email /bin/
COPY --from=cordial-build /cordial/lib/libemail.so /lib/
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# build a UBI9
FROM redhat/ubi9 AS cordial-run-ubi9
COPY --from=cordial-build /cordial/bin/geneos /bin/
COPY --from=cordial-build /cordial/bin/gdna /bin/
COPY --from=cordial-build /cordial/bin/gateway-reporter /bin/
COPY --from=cordial-build /cordial/bin/dv2email /bin/
COPY --from=cordial-build /cordial/lib/libemail.so /lib/
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# build a centos7 image for testing
FROM centos:7 AS cordial-run-centos7
COPY --from=cordial-build /cordial/bin/geneos /bin/
COPY --from=cordial-build /cordial/bin/gateway-reporter /bin/
COPY --from=cordial-build /cordial/bin/dv2email /bin/
COPY --from=cordial-build /cordial/lib/libemail.so /lib/
RUN --mount=type=cache,target=/var/rpm \
    set -eux; \
    yum update -y; \
    yum install -y \
        fontconfig \
        ca-certificates; \
    useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# create a runnable gdna test image using basic debian
FROM debian:stable AS gdna

COPY --from=build /app/cordial/tools/geneos/geneos /bin/

COPY --from=build /app/cordial/gdna/gdna /bin/
COPY --from=build /app/cordial/gdna/cmd/gdna.defaults.yaml /etc/geneos/
COPY --from=build /app/cordial/gdna/gdna.yaml /etc/geneos/
COPY --from=build /app/cordial/gdna/geneos/* /etc/geneos/gdna/

COPY --chmod=555 gdna/docker/start-up.sh /etc/geneos/gdna/start-up.sh

ENV DEBIANFRONTEND=noninteractive
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    set -eux; \
    apt update; \
    apt install -y --no-install-recommends \
        ca-certificates \
        fontconfig \
        fonts-open-sans \
        sqlite3 \
        ; \
    rm -fr /var/lib/apt/lists/*; \
    chmod +x /bin/geneos; \
    useradd -ms /bin/bash geneos

WORKDIR /home/geneos
USER geneos

#
# to install from locally downloaded release archives, use this mount
# and add `--archive /downloads` to each `geneos deploy` line
#
# RUN --mount=source=downloads,target=/downloads \
#
RUN mkdir -p /home/geneos/.config/geneos
RUN --mount=type=secret,id=credentials.json,mode=0444,required,target=/home/geneos/.config/geneos/credentials.json \
    --mount=type=secret,id=keyfile.aes,mode=0444,required,target=/home/geneos/.config/geneos/keyfile.aes \
    set -eux; \
    mkdir gdna; \
    geneos deploy gateway "Demo Gateway" --geneos /home/geneos --nosave --port 8100 --tls options="-demo" --include 100:/etc/geneos/gdna/gdna.include.xml; \
    geneos deploy netprobe GDNA --nosave --port 8101; \
    geneos deploy webserver GDNA --nosave --port 8443 --import config/config.xml=/etc/geneos/gdna/web-config.xml;
ENTRYPOINT [ "/etc/geneos/gdna/start-up.sh" ]
CMD [ "bash" ]
