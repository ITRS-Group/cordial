# Dockerfile to build cordial components and tar.gz files

ARG GOVERSION=1.24.3

# The bullseye image seems to offer the most compatibility, including
# libemail.so dependencies
#
# Note: To build an executable for a modern Mac use something like:
#
# GOOS=darwin GOARCH=arm64 go build -o geneos.${GOOS}-${GOARCH} -tags netgo,osusergo --ldflags '-s -w'
# 
FROM golang:${GOVERSION}-bullseye AS build
# base files
COPY go.mod go.sum cordial.go logging.go VERSION README.md CHANGELOG.md /app/cordial/
COPY pkg /app/cordial/pkg
# geneos, dv2email, gateway-reporter, san-config
COPY tools /app/cordial/tools
# servicenow*, pagerduty
COPY integrations /app/cordial/integrations/
# libemail, libalerts
COPY libraries /app/cordial/libraries/
# gdna
COPY gdna /app/cordial/gdna
RUN --mount=type=cache,target=/go/pkg/mod \
    set -eux; \
    # build geneos (in Windows version)
    cd /app/cordial/tools/geneos; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build --ldflags '-s -w'; \
    # gateway-reporter (in Windows version)
    cd /app/cordial/tools/gateway-reporter; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build --ldflags '-s -w'; \
    # dv2email
    cd /app/cordial/tools/dv2email; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # san-config (inc Windows version)
    cd /app/cordial/tools/san-config; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build --ldflags '-s -w'; \
    # servicenow
    cd /app/cordial/integrations/servicenow; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # servicenow2
    cd /app/cordial/integrations/servicenow2; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # pagerduty
    cd /app/cordial/integrations/pagerduty; \
    go build -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # gdna (inc Windows version)
    cd /app/cordial/gdna; \
    go build -tags sqlite_omit_load_extension,netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build --ldflags '-s -w';

# build non-static binaries (shared libs) with UBI8 for GLIBC backward compatibility
FROM redhat/ubi8 AS build-ubi8
RUN set -eux; \
    yum -y install gcc make golang
ARG GOVERSION
ENV GOTOOLCHAIN=go${GOVERSION}
# base files
COPY go.mod go.sum cordial.go logging.go VERSION README.md CHANGELOG.md /app/cordial/
COPY pkg /app/cordial/pkg
# geneos, dv2email, gateway-reporter, san-config
COPY tools /app/cordial/tools
# servicenow*, pagerduty
COPY integrations /app/cordial/integrations/
# libemail, libalerts
COPY libraries /app/cordial/libraries/
# gdna
COPY gdna /app/cordial/gdna
RUN --mount=type=cache,target=/go/pkg/mod \
    set -eux; \
    # libemail
    cd /app/cordial/libraries/libemail; \
    make; \
    # libalerts
    cd /app/cordial/libraries/libalert; \
    make

#
# Build PDF documentation using mdpdf. Like all Puppeteer based PDF
# writers the support for relative links to files is broken, so
# documents with links to other docs in the same repo will be wrong.
#
FROM node:lts AS cordial-docs
ARG NODE_ENV=production
ENV NODE_ENV $NODE_ENV
ENV PUPPETEER_SKIP_DOWNLOAD true
ENV PUPPETEER_EXECUTABLE_PATH /usr/bin/google-chrome-stable
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    set -eux; \
    apt update; \
    apt install -y --no-install-recommends \
        wget \
        gnupg; \
    wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add -; \
    sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list'; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        fonts-noto-color-emoji \
        fonts-ipafont-gothic \
        fonts-wqy-zenhei \
        fonts-thai-tlwg \
        fonts-kacst \
        fonts-freefont-ttf \
        google-chrome-stable \
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
        libasound2 \
        libxss1; \
    rm -fr /var/lib/apt/lists/*;
RUN set -eux; \
    mkdir node_modules; \
    npm install --verbose --global git+https://github.com/elliotblackburn/mdpdf.git#3.0.4; \
    npm install --global @mermaid-js/mermaid-cli

# base files
COPY VERSION README.md CHANGELOG.md /app/cordial/
COPY pkg /app/cordial/pkg
# geneos, dv2email, gateway-reporter, san-config
COPY tools /app/cordial/tools
# servicenow*, pagerduty
COPY integrations /app/cordial/integrations/
# libemail, libalerts
COPY libraries /app/cordial/libraries/
# gdna
COPY gdna /app/cordial/gdna

WORKDIR /app/cordial/doc-output
COPY tools/geneos/README.md geneos.md
COPY tools/gateway-reporter/README.md gateway-reporter.md
COPY tools/dv2email/README.md dv2email.md
COPY tools/dv2email/screenshots/ screenshots/
COPY tools/san-config/README.md san-config.md
COPY gdna/*.md gdna/
COPY gdna/screenshots/ screenshots/
COPY integrations/servicenow/README.md servicenow.md
COPY integrations/servicenow2/README.md servicenow2.md
COPY integrations/pagerduty/README.md pagerduty.md
COPY libraries/libemail/README.md libemail.md
COPY libraries/libalert/README.md libalert.md

ARG MERMAID=".mermaid"
ARG READMEDIRS="tools/geneos tools/gateway-reporter tools/dv2email tools/san-config integrations/servicenow integrations/servicenow2 integrations/pagerduty libraries/libemail libraries/libalert"
RUN set -eux; \
    echo '{ "args": ["--no-sandbox"] }' > /puppeteer.json; \
    for i in ${READMEDIRS}; \
    do \
            mmdc -p /puppeteer.json -i /app/cordial/$i/README.md -o /app/cordial/$i/README${MERMAID}.md; \
            mdpdf /app/cordial/$i/README${MERMAID}.md ${i##*/}.pdf --border=15mm --gh-style; \
    done; \
    for i in gdna/*.md; \
    do \
            b=$(basename ${i%.md}); \
            mmdc -p /puppeteer.json -i /app/cordial/$i -o /app/cordial/gdna/${b}${MERMAID}.md; \
            mdpdf /app/cordial/gdna/${b}${MERMAID}.md gdna/gdna-${b##*/}.pdf --border=15mm --gh-style; \
            mv gdna/$b.md gdna/gdna-$b.md; \
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

COPY --from=build /app/cordial/VERSION /
COPY --from=build /app/cordial/CHANGELOG.md /cordial/docs/
COPY --from=build /app/cordial/README.md /cordial/docs/

# tools binaries
COPY --from=build /app/cordial/tools/geneos/geneos /cordial/bin/
COPY --from=build /app/cordial/tools/geneos/geneos.exe /cordial/bin/
COPY --from=build /app/cordial/tools/gateway-reporter/gateway-reporter /cordial/bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /cordial/bin/

# san-config and default YAML
COPY --from=build /app/cordial/tools/san-config/san-config /cordial/bin/
COPY --from=build /app/cordial/tools/san-config/cmd/san-config.defaults.yaml /cordial/etc/geneos/

# docs
COPY --from=cordial-docs /app/cordial/doc-output /cordial/docs/

# servicenow
COPY --from=build /app/cordial/integrations/servicenow/servicenow /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow/servicenow.example.yaml /cordial/etc/geneos/

# servicenow2
COPY --from=build /app/cordial/integrations/servicenow2/servicenow2 /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow2/servicenow2.client.example.yaml /cordial/etc/geneos/
COPY --from=build /app/cordial/integrations/servicenow2/servicenow2.router.example.yaml /cordial/etc/geneos/

# pagerduty
COPY --from=build /app/cordial/integrations/pagerduty/pagerduty /cordial/bin/
COPY --from=build /app/cordial/integrations/pagerduty/cmd/pagerduty.defaults.yaml /cordial/etc/geneos/

# libemail / libalerts
COPY --from=build-ubi8 /app/cordial/libraries/libemail/libemail.so /cordial/lib/
COPY --from=build-ubi8 /app/cordial/libraries/libalert/libalert.so /cordial/lib/

# gdna
COPY --from=build /app/cordial/gdna/gdna /cordial/bin/
COPY --from=build /app/cordial/gdna/gdna.exe /cordial/bin/
COPY --from=build /app/cordial/gdna/cmd/gdna.defaults.yaml /cordial/etc/geneos/
COPY --from=build /app/cordial/gdna/gdna.example.yaml /cordial/etc/geneos/gdna.example.yaml
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
FROM ubuntu:jammy AS cordial-run-ubuntu
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/gateway-reporter/gateway-reporter /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build /app/cordial/tools/san-config/san-config /bin/
COPY --from=build /app/cordial/libraries/libemail/libemail.so /lib/
COPY --from=build /app/cordial/gdna/gdna /bin/
RUN --mount=type=cache,target=/var/cache/apt \
    set -eux; \
    apt update; \
    apt install -y --no-install-recommends \
    fontconfig \
    ca-certificates \
    openjdk-21-jre-headless \
    ; \
    useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# build a UBI8 for testing
FROM redhat/ubi8 AS cordial-run-ubi8
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/gateway-reporter/gateway-reporter /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build /app/cordial/tools/san-config/san-config /bin/
COPY --from=build /app/cordial/libraries/libemail/libemail.so /lib/
COPY --from=build /app/cordial/gdna/gdna /bin/
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# build a UBI9
FROM redhat/ubi9 AS cordial-run-ubi9
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/gateway-reporter/gateway-reporter /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build /app/cordial/libraries/libemail/libemail.so /lib/
COPY --from=build /app/cordial/gdna/gdna /bin/
RUN set -eux; \
    dnf install -y libnsl2 \
    ; \
    dnf clean all \
    ;
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# create a runnable gdna test image using basic debian
FROM debian:stable AS gdna

COPY --from=build /app/cordial/tools/geneos/geneos /bin/

COPY --from=build /app/cordial/gdna/gdna /bin/
COPY --from=build /app/cordial/gdna/cmd/gdna.defaults.yaml /etc/geneos/
COPY --from=build /app/cordial/gdna/gdna.example.yaml /etc/geneos/
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
RUN --mount=type=secret,id=credentials.json,mode=0444,uid=1000,required,target=/home/geneos/.config/geneos/credentials.json \
    --mount=type=secret,id=keyfile.aes,mode=0444,uid=1000,required,target=/home/geneos/.config/geneos/keyfile.aes \
    set -eux; \
    mkdir gdna; \
    geneos deploy gateway "Demo Gateway" --geneos /home/geneos --nosave --port 8100 --tls options="-demo" --include 100:/etc/geneos/gdna/gdna.include.xml; \
    geneos deploy netprobe minimal:GDNA --nosave --port 8101; \
    geneos deploy webserver GDNA --nosave --port 8443 --import config/config.xml=/etc/geneos/gdna/web-config.xml;
ENTRYPOINT [ "/etc/geneos/gdna/start-up.sh" ]
CMD [ "bash" ]
