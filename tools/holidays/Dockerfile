
FROM python:3.7
ARG GOVERSION=1.21.0
ARG BUILDOS=linux
ARG BUILDARCH=amd64
RUN set -eux; \
    # apk add --no-cache \
    #   build-base \
    #   pkgconfig \
    #   upx \
    apt update; \
    apt install -y --no-install-recommends \
        build-essential \
        pkg-config \
    ; \
    pip3 install holidays
ADD https://go.dev/dl/go${GOVERSION}.${BUILDOS}-${BUILDARCH}.tar.gz /tmp/
RUN tar -C /usr/local -xzf /tmp/go${GOVERSION}.${BUILDOS}-${BUILDARCH}.tar.gz
ENV PATH=$PATH:/usr/local/go/bin
COPY main.go *.py /app/holidays/
COPY cmd /app/holidays/cmd
COPY go.mod.txt /app/holidays/go.mod
    
RUN set -eux; \
    cd /app/holidays; \
    go mod tidy; \
    go build --ldflags '-s -w'

CMD /bin/sh