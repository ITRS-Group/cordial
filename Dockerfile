# Dockerfile to build cordial components and tar.gz files

# To build a release with docker images for testing, you must create a
# credentials.json file in your user config directory, like this:
#
# $ geneos login -u user@example.com
#
# This is to allow downloading Geneos release archives during the build.

ARG GOVERSION=1.27.4

# Note: To build an executable for a modern Mac use something like:
#
# GOOS=darwin GOARCH=arm64 go build -trimpath -o geneos.${GOOS}-${GOARCH} -tags netgo,osusergo --ldflags '-s -w'
#
# However without executable signing this will not run on a Mac unless built locally.

FROM golang:${GOVERSION} AS build
WORKDIR /app/cordial
ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/root/.cache/go-build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download -x
# base files
COPY cordial.go VERSION README.md CHANGELOG.md UPDATENOTES.md HIGHLIGHTS.md /app/cordial/
COPY pkg /app/cordial/pkg
# tools
COPY tools /app/cordial/tools
# integrations
COPY integrations /app/cordial/integrations/
# libemail, libalerts
COPY libraries /app/cordial/libraries/
# gdna
COPY gdna /app/cordial/gdna
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target="/root/.cache/go-build" \
    set -eux; \
    # build geneos (in Windows version)
    cd /app/cordial/tools/geneos; \
    go build -trimpath -tags netgo,osusergo,noxlsx --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build -trimpath -tags noxlsx --ldflags '-s -w'; \
    cd /app/cordial/tools/ims-gateway; \
    go build -trimpath -tags netgo,osusergo,noxlsx --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build -trimpath -tags noxlsx --ldflags '-s -w'; \
    # dv2email
    cd /app/cordial/tools/dv2email; \
    go build -trimpath -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # servicenow2
    cd /app/cordial/integrations/servicenow2; \
    go build -trimpath -tags netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    # gdna (inc Windows version)
    cd /app/cordial/gdna; \
    go build -trimpath -tags sqlite_omit_load_extension,netgo,osusergo --ldflags '-s -w -linkmode external -extldflags=-static'; \
    GOOS=windows go build -trimpath --ldflags '-s -w';

# build non-static binaries (shared libs) with UBI8 for GLIBC backward compatibility
FROM redhat/ubi8 AS build-ubi8
RUN --mount=type=cache,id=ubi8-dnf,target=/var/cache/dnf,sharing=locked \
    set -eux; \
    yum -y --setopt=keepcache=True install gcc make golang
ARG GOVERSION
# the distro Go is too old for this module, and is only used to download
# the toolchain named here (which needs network access) that does the
# build; do not drop the pin as it keeps the same Go version as the
# build stage
ENV GOTOOLCHAIN=go${GOVERSION}
ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/root/.cache/go-build
WORKDIR /app/cordial
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download -x
# base files
COPY cordial.go VERSION README.md CHANGELOG.md UPDATENOTES.md HIGHLIGHTS.md /app/cordial/
COPY pkg /app/cordial/pkg
# tools
COPY tools /app/cordial/tools
# integrations
COPY integrations /app/cordial/integrations/
# libemail, libalerts
COPY libraries /app/cordial/libraries/
# gdna
COPY gdna /app/cordial/gdna
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target="/root/.cache/go-build" \
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
# utils/pdf supplies the shared print stylesheet and GitHub alert preprocessing.
# qpdf losslessly compresses the generated PDFs without producing PDF/A.
#
FROM node:24 AS cordial-docs
ARG NODE_ENV=production
ENV NODE_ENV=$NODE_ENV
ENV PUPPETEER_SKIP_DOWNLOAD=true
ENV PUPPETEER_EXECUTABLE_PATH=/usr/bin/google-chrome-stable
RUN --mount=type=cache,id=docs-apt-cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,id=docs-apt-lists,target=/var/lib/apt/lists,sharing=locked \
    set -eux; \
    rm -f /etc/apt/apt.conf.d/docker-clean; \
    apt-get update; \
    apt-get -o APT::Keep-Downloaded-Packages=true install -y --no-install-recommends \
        wget \
        gnupg; \
    install -d -m 0755 /etc/apt/keyrings; \
    wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | gpg --dearmor -o /etc/apt/keyrings/google-chrome.gpg; \
    echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/google-chrome.gpg] https://dl.google.com/linux/chrome/deb/ stable main" > /etc/apt/sources.list.d/google.list; \
    apt-get update; \
    apt-get -o APT::Keep-Downloaded-Packages=true install -y --no-install-recommends \
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
        libxss1;
RUN --mount=type=cache,target=/root/.npm,sharing=locked \
    set -eux; \
    mkdir node_modules; \
    npm install --prefer-offline --verbose --global git+https://github.com/elliotblackburn/mdpdf.git#3.0.4; \
    npm install --prefer-offline --global @mermaid-js/mermaid-cli
RUN --mount=type=cache,id=docs-apt-cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,id=docs-apt-lists,target=/var/lib/apt/lists,sharing=locked \
    apt-get -o APT::Keep-Downloaded-Packages=true install -y --no-install-recommends fonts-dejavu-core qpdf
# Barlow is used on the ITRS documentation site and licensed under the SIL OFL.
RUN set -eux; \
    mkdir -p /usr/local/share/fonts/barlow; \
    for file in Barlow-Regular.ttf Barlow-Italic.ttf Barlow-Bold.ttf Barlow-BoldItalic.ttf OFL.txt; do \
        wget -q -O "/usr/local/share/fonts/barlow/$file" \
            "https://raw.githubusercontent.com/google/fonts/89f5431ff0db41bd2fe3f7ba21a723a01622428b/ofl/barlow/$file"; \
    done; \
    fc-cache -f /usr/local/share/fonts/barlow
RUN --mount=type=cache,target=/root/.npm,sharing=locked \
    npm install --prefer-offline --global markdown-it@14.1.0
ENV NODE_PATH=/usr/local/lib/node_modules
ENV MDPDF_STYLES=/app/pdf/print.css

# base files
COPY utils/pdf/ /app/pdf/
COPY VERSION README.md CHANGELOG.md UPDATENOTES.md HIGHLIGHTS.md /app/cordial/
COPY pkg /app/cordial/pkg
# tools
COPY tools /app/cordial/tools
# integrations
COPY integrations /app/cordial/integrations/
# libemail, libalerts
COPY libraries /app/cordial/libraries/
# gdna
COPY gdna /app/cordial/gdna

WORKDIR /docs
RUN cp /usr/local/share/fonts/barlow/OFL.txt Barlow-OFL.txt
COPY tools/ims-gateway/*.md ims-gateway/
COPY tools/ims-gateway/screenshots/ ims-gateway/screenshots/
COPY tools/dv2email/README.md dv2email.md
COPY tools/dv2email/screenshots/ screenshots/
COPY gdna/*.md gdna/
COPY gdna/screenshots/ screenshots/
COPY integrations/servicenow2/README.md servicenow2.md
COPY libraries/libemail/README.md libemail.md
COPY libraries/libalert/README.md libalert.md

ARG MERMAID=".mermaid"
ARG READMEDIRS="tools/dv2email integrations/servicenow2 libraries/libemail libraries/libalert"
RUN set -eux; \
    rm -f /app/cordial/tools/geneos/DOCUMENTATION.md /app/cordial/tools/geneos/NOTES.md /app/cordial/tools/geneos/IMS.md /app/cordial/tools/geneos/INSTALL.md; \
    echo '{ "args": ["--no-sandbox"] }' > /puppeteer.json; \
    mkdir -p geneos; \
    for i in /app/cordial/tools/geneos/*.md; \
    do \
            b=$(basename "$i" .md); \
            case "$b" in AGENTS|NOTES|DOCUMENTATION|IMS|INSTALL) continue ;; esac; \
            name=$(printf '%s' "$b" | tr '[:upper:]_' '[:lower:]-'); \
            destination="geneos-${name}.md"; \
            if [ "$b" = README ]; then destination=README.md; fi; \
            cp "$i" "geneos/$destination"; \
        done; \
        for i in /app/cordial/tools/geneos/*.md; \
        do \
            b=$(basename "$i" .md); \
            case "$b" in AGENTS|NOTES|DOCUMENTATION|IMS|INSTALL) continue ;; esac; \
            name=$(printf '%s' "$b" | tr '[:upper:]_' '[:lower:]-'); \
            destination="geneos-${name}.md"; \
            if [ "$b" = README ]; then destination=README.md; fi; \
            for target in geneos/*.md; \
            do \
                sed -i "s#(${b}\\.md#(${destination}#g" "$target"; \
            done; \
    done; \
        sed -i 's#(docs/geneos.md#(geneos-commands.md#g' geneos/*.md; \
        for i in geneos/*.md; \
        do \
            b=$(basename "$i" .md); \
            mmdc -p /puppeteer.json -i "$i" -o "/tmp/${b}${MERMAID}.md"; \
            node /app/pdf/mdpdf.cjs "/tmp/${b}${MERMAID}.md" "${i%.md}.pdf" --border=15mm --gh-style; \
        done; \
            for i in ims-gateway/*.md; \
            do \
                b=$(basename "$i" .md); \
                if [ "$b" = README ]; then \
                    destination=README.md; \
                    output=README.pdf; \
                else \
                    name=$(printf '%s' "$b" | tr '[:upper:]_' '[:lower:]-'); \
                    destination="ims-gateway-${name}.md"; \
                    output="ims-gateway-${name}.pdf"; \
                    mv "$i" "ims-gateway/$destination"; \
                fi; \
                if [ "$b" = README ]; then \
                    sed -i 's#(SDP-AUTH.md#(ims-gateway-sdp-auth.md#g' "ims-gateway/$destination"; \
                fi; \
                mmdc -p /puppeteer.json -i "/app/cordial/tools/ims-gateway/${b}.md" -o "/app/cordial/tools/ims-gateway/${b}${MERMAID}.md"; \
                node /app/pdf/mdpdf.cjs "/app/cordial/tools/ims-gateway/${b}${MERMAID}.md" "ims-gateway/$output" --border=15mm --gh-style; \
            done; \
    for i in ${READMEDIRS}; \
    do \
            mmdc -p /puppeteer.json -i /app/cordial/$i/README.md -o /app/cordial/$i/README${MERMAID}.md; \
            node /app/pdf/mdpdf.cjs /app/cordial/$i/README${MERMAID}.md ${i##*/}.pdf --border=15mm --gh-style; \
    done; \
    for i in gdna/*.md; \
    do \
            b=$(basename "$i" .md); \
            name=$(printf '%s' "$b" | tr '[:upper:]_' '[:lower:]-'); \
            mmdc -p /puppeteer.json -i "/app/cordial/gdna/${b}.md" -o "/app/cordial/gdna/${b}${MERMAID}.md"; \
            node /app/pdf/mdpdf.cjs "/app/cordial/gdna/${b}${MERMAID}.md" "gdna/gdna-${name}.pdf" --border=15mm --gh-style; \
            mv "$i" "gdna/gdna-${name}.md"; \
    done; \
        for target in gdna/gdna-*.md; \
        do \
            for source in /app/cordial/gdna/*.md; \
            do \
                b=$(basename "$source" .md); \
                name=$(printf '%s' "$b" | tr '[:upper:]_' '[:lower:]-'); \
                sed -i "s#(${b}\\.md#(gdna-${name}.md#g" "$target"; \
            done; \
        done; \
    cp /app/cordial/tools/geneos/docs/COMMANDS.md geneos/geneos-commands.md; \
    mmdc -p /puppeteer.json -i /app/cordial/tools/geneos/docs/COMMANDS.md -o /tmp/geneos-commands${MERMAID}.md; \
    sed 's#^---$#<div style="break-before: page; page-break-before: always;"></div>#' /tmp/geneos-commands${MERMAID}.md > /tmp/geneos-commands-pdf.md; \
    node /app/pdf/mdpdf.cjs /tmp/geneos-commands-pdf.md geneos/geneos-commands.pdf --border=15mm --gh-style; \
    for name in CHANGELOG UPDATENOTES HIGHLIGHTS; \
    do \
        cp "/app/cordial/$name.md" "$name.md"; \
    done; \
    sed -i 's#(tools/geneos/TLS.md)#(geneos/geneos-tls.md)#g' UPDATENOTES.md; \
    for name in CHANGELOG UPDATENOTES HIGHLIGHTS; \
    do \
        mmdc -p /puppeteer.json -i "$name.md" -o "/tmp/${name}${MERMAID}.md"; \
        node /app/pdf/mdpdf.cjs "/tmp/${name}${MERMAID}.md" "$name.pdf" --border=15mm --gh-style; \
    done; \
    cp /app/cordial/README.md README.md; \
    mmdc -p /puppeteer.json -i /app/cordial/README.md -o /app/cordial/README${MERMAID}.md; \
    node /app/pdf/mdpdf.cjs /app/cordial/README${MERMAID}.md README.pdf --border=15mm --gh-style

RUN set -eux; \
    mkdir -p geneos-includes geneos-holidays-include; \
    for source in /app/cordial/pkg/includes/README.md /app/cordial/pkg/includes/_docs/*.md; \
    do \
        name=$(basename "$source" .md); \
        cp "$source" "geneos-includes/$name.md"; \
        sed -i -E \
            -e 's#\]\((\.\./){3}([^)]*)\)#](https://github.com/ITRS-Group/cordial/blob/main/\2)#g' \
            "geneos-includes/$name.md"; \
        mmdc -p /puppeteer.json -i "geneos-includes/$name.md" -o "/tmp/${name}.mermaid.md"; \
        node /app/pdf/mdpdf.cjs "/tmp/${name}.mermaid.md" "geneos-includes/$name.pdf" --border=15mm --gh-style; \
    done; \
    for source in /app/cordial/pkg/holidays/README.md /app/cordial/pkg/holidays/RUNBOOK.md /app/cordial/pkg/holidays/holiday-calendars-*.md; \
    do \
        name=$(basename "$source" .md); \
        cp "$source" "geneos-holidays-include/$name.md"; \
        mmdc -p /puppeteer.json -i "$source" -o "/tmp/holidays-${name}.mermaid.md"; \
        node /app/pdf/mdpdf.cjs "/tmp/holidays-${name}.mermaid.md" "geneos-holidays-include/$name.pdf" --border=15mm --gh-style; \
    done

RUN set -eux; \
    find /docs -type f -name '*.pdf' -exec sh -ec ' \
        for pdf do \
            tmp=$(mktemp "${pdf}.XXXXXXXX"); \
            trap '\''rm -f -- "$tmp"'\'' EXIT; \
            qpdf --object-streams=generate --recompress-flate --compression-level=9 "$pdf" "$tmp"; \
            if [ "$(wc -c < "$tmp")" -lt "$(wc -c < "$pdf")" ]; then \
                chmod --reference="$pdf" "$tmp"; \
                mv -f -- "$tmp" "$pdf"; \
            else \
                rm -f -- "$tmp"; \
            fi; \
            trap - EXIT; \
        done' sh {} +

#
# assemble files from previous stages into a .tar.gz ready for extraction in the
# Makefile
#
FROM alpine AS geneos-includes

RUN --mount=type=cache,id=alpine-apk,target=/var/cache/apk,sharing=locked \
    apk add tar

COPY pkg/includes/VERSION /VERSION
COPY pkg/includes/files/itrs.*.xml /includes-source/
COPY pkg/includes/files/config.*.xml /includes-source/
COPY pkg/includes/scripts/install-includes.sh /includes-source/scripts/install-includes.sh
COPY --from=cordial-docs /docs/geneos-includes/ /includes-source/docs/
RUN set -eux; \
    version=$(cat /VERSION); \
    archive_root="geneos-includes-$version"; \
    mkdir -p "/includes/$archive_root/itrs-$version" "/includes/$archive_root/config" "/includes/$archive_root/scripts" "/includes/$archive_root/docs"; \
    cp /includes-source/itrs.*.xml "/includes/$archive_root/itrs-$version/"; \
    cp /includes-source/config.*.xml "/includes/$archive_root/config/"; \
    cp /includes-source/scripts/install-includes.sh "/includes/$archive_root/scripts/"; \
    cp /includes-source/docs/* "/includes/$archive_root/docs/"; \
    find /includes -type f -exec chmod a-x {} \;; \
    find "/includes/$archive_root/itrs-$version" -type f -exec chmod a-w {} \;; \
    chmod 0555 "/includes/$archive_root/scripts/install-includes.sh"; \
    tar --sort=name -czf "/geneos-includes-$version.tar.gz" -C /includes "$archive_root"

FROM alpine AS geneos-holidays-include

RUN --mount=type=cache,id=alpine-apk,target=/var/cache/apk,sharing=locked \
    apk add tar

COPY pkg/holidays/VERSION /VERSION
COPY pkg/holidays/files/holidays.*.xml /holidays-source/
COPY pkg/includes/scripts/install-includes.sh /holidays-source/scripts/install-includes.sh
COPY --from=cordial-docs /docs/geneos-holidays-include/ /holidays-source/docs/
RUN set -eux; \
    version=$(cat /VERSION); \
    archive_root="geneos-includes-$version"; \
    mkdir -p "/holidays/$archive_root/holidays-$version" "/holidays/$archive_root/scripts" "/holidays/$archive_root/docs"; \
    cp /holidays-source/holidays.*.xml "/holidays/$archive_root/holidays-$version/"; \
    cp /holidays-source/scripts/install-includes.sh "/holidays/$archive_root/scripts/"; \
    cp /holidays-source/docs/* "/holidays/$archive_root/docs/"; \
    find /holidays -type f -exec chmod a-x {} \;; \
    find "/holidays/$archive_root/holidays-$version" -type f -exec chmod a-w {} \;; \
    chmod 0555 "/holidays/$archive_root/scripts/install-includes.sh"; \
    tar --sort=name -czf "/geneos-holiday-includes-$version.tar.gz" -C /holidays "$archive_root"

FROM alpine AS cordial-build

RUN --mount=type=cache,id=alpine-apk,target=/var/cache/apk,sharing=locked \
    apk add tar

COPY --from=build /app/cordial/VERSION /
COPY --from=cordial-docs /docs/. /cordial/docs/
COPY --from=geneos-includes /geneos-includes-*.tar.gz /
COPY --from=geneos-holidays-include /geneos-holiday-includes-*.tar.gz /

# tools binaries
COPY --from=build /app/cordial/tools/geneos/geneos /cordial/bin/
COPY --from=build /app/cordial/tools/geneos/geneos.exe /cordial/bin/
COPY --from=build /app/cordial/tools/ims-gateway/ims-gateway /cordial/bin/
COPY --from=build /app/cordial/tools/ims-gateway/ims-gateway.exe /cordial/bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /cordial/bin/
COPY --from=build /app/cordial/tools/dv2email/cmd/dv2email.defaults.yaml /cordial/etc/geneos/dv2email.example.yaml

# servicenow2
COPY --from=build /app/cordial/integrations/servicenow2/servicenow2 /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow2/servicenow2.client.example.yaml /cordial/etc/geneos/
COPY --from=build /app/cordial/integrations/servicenow2/servicenow2.proxy.example.yaml /cordial/etc/geneos/

# libemail / libalerts
COPY --from=build-ubi8 /app/cordial/libraries/libemail/libemail.so /cordial/lib/
COPY --from=build-ubi8 /app/cordial/libraries/libalert/libalert.so /cordial/lib/

# gdna
COPY --from=build /app/cordial/gdna/gdna /cordial/bin/
COPY --from=build /app/cordial/gdna/gdna.exe /cordial/bin/
COPY --from=build /app/cordial/gdna/cmd/gdna.defaults.yaml /cordial/etc/geneos/
COPY --from=build /app/cordial/gdna/gdna.example.yaml /cordial/etc/geneos/gdna.example.yaml
COPY --from=build /app/cordial/gdna/geneos/* /cordial/etc/geneos/gdna/
# don't copy *.history that is created on build (delete it afterwards)
RUN rm -f /cordial/etc/geneos/gdna/*.history

# build tar
# upx -q /cordial/bin/*; \
RUN set -eux; \
    cp -r /cordial /cordial-$(cat /VERSION); \
    tar --sort=name -czf /cordial-$(cat /VERSION).tar.gz cordial-$(cat /VERSION)

CMD [ "bash" ]

# hand testing images

# create a runnable test image using basic debian
# FROM ubuntu:jammy AS cordial-run-ubuntu
FROM debian:stable AS cordial-run-debian
RUN --mount=type=cache,id=debian-apt-cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,id=debian-apt-lists,target=/var/lib/apt/lists,sharing=locked \
    set -eux; \
    rm -f /etc/apt/apt.conf.d/docker-clean; \
    apt-get update; \
    apt-get -o APT::Keep-Downloaded-Packages=true install -y --no-install-recommends \
    fontconfig \
    ca-certificates \
    openjdk-21-jre-headless \
    ; \
    useradd -ms /bin/bash geneos
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build-ubi8 /app/cordial/libraries/libemail/libemail.so /lib/
COPY --from=build /app/cordial/gdna/gdna /bin/
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# build a UBI8 for testing
FROM redhat/ubi8 AS cordial-run-ubi8
RUN --mount=type=cache,id=ubi8-dnf,target=/var/cache/dnf,sharing=locked \
    set -eux; \
    dnf install -y --setopt=keepcache=True libnsl2 fontconfig \
    ; \
    ln -s /usr/lib64/libnsl.so.2 /usr/lib64/libnsl.so.1
RUN useradd -ms /bin/bash geneos
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build-ubi8 /app/cordial/libraries/libemail/libemail.so /lib/
COPY --from=build /app/cordial/gdna/gdna /bin/
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# build a UBI9
FROM redhat/ubi9 AS cordial-run-ubi9
RUN --mount=type=cache,id=ubi9-dnf,target=/var/cache/dnf,sharing=locked \
    dnf install -y --setopt=keepcache=True fontconfig
RUN useradd -ms /bin/bash geneos
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build-ubi8 /app/cordial/libraries/libemail/libemail.so /lib/
COPY --from=build /app/cordial/gdna/gdna /bin/
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# build a UBI10
FROM redhat/ubi10 AS cordial-run-ubi10
RUN --mount=type=cache,id=ubi10-dnf,target=/var/cache/dnf,sharing=locked \
    dnf install -y --setopt=keepcache=True libnsl libxcrypt-compat fontconfig
RUN useradd -ms /bin/bash geneos
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/tools/dv2email/dv2email /bin/
COPY --from=build-ubi8 /app/cordial/libraries/libemail/libemail.so /lib/
COPY --from=build /app/cordial/gdna/gdna /bin/
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]

# create a runnable gdna test image using basic debian
FROM debian:stable AS gdna

ENV DEBIAN_FRONTEND=noninteractive
ENV ITRS_DOWNLOAD_TIMEOUT=5m
RUN --mount=type=cache,id=debian-apt-cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,id=debian-apt-lists,target=/var/lib/apt/lists,sharing=locked \
    set -eux; \
    rm -f /etc/apt/apt.conf.d/docker-clean; \
    apt-get update; \
    apt-get -o APT::Keep-Downloaded-Packages=true install -y --no-install-recommends \
        ca-certificates \
        fontconfig \
        fonts-open-sans \
        sqlite3 \
        ; \
    useradd -ms /bin/bash geneos

COPY --chmod=755 --from=build /app/cordial/tools/geneos/geneos /bin/
COPY --from=build /app/cordial/gdna/gdna /bin/
COPY --from=build /app/cordial/gdna/cmd/gdna.defaults.yaml /etc/geneos/
COPY --from=build /app/cordial/gdna/gdna.example.yaml /etc/geneos/
COPY --from=build /app/cordial/gdna/geneos/* /etc/geneos/gdna/
COPY --chmod=555 gdna/docker/start-up.sh /etc/geneos/gdna/start-up.sh

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
    geneos deploy gateway "Demo Gateway" --geneos /home/geneos --nosave --port 8100 options="-demo" --include 100:/etc/geneos/gdna/gdna.include.xml; \
    geneos deploy netprobe minimal:GDNA --nosave --port 8101; \
    geneos deploy webserver GDNA --nosave --port 8443 --import config/config.xml=/etc/geneos/gdna/web-config.xml;
ENTRYPOINT [ "/etc/geneos/gdna/start-up.sh" ]
CMD [ "bash" ]
