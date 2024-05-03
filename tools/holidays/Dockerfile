FROM debian:buster
ARG PYVERSION=3.7.17
ARG GOVERSION=1.21.0
ARG BUILDOS=linux
ARG BUILDARCH=amd64
RUN set -eux; \
    apt update -y --no-install-recommends; \
    apt install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        python3.7 \
        python3.7-dev \
        python3-pip \
        pkg-config \
    ;

# FROM alpine
# ARG PYVERSION=3.7.17
# ARG GOVERSION=1.21.0
# ARG BUILDOS=linux
# ARG BUILDARCH=amd64
# RUN set -eux; \
#     apk add --no-cache \
#       build-base \
#       pkgconfig \
#       upx \
#     ; \
#     apk add --no-cache --virtual .build-deps \
#         bluez-dev \
#         bzip2-dev \
#         coreutils \
#         dpkg-dev dpkg \
#         expat-dev \
#         findutils \
#         gcc \
#         gdbm-dev \
#         libc-dev \
#         libffi-dev \
#         libnsl-dev \
#         libtirpc-dev \
#         linux-headers \
#         make \
#         ncurses-dev \
#         openssl-dev \
#         pax-utils \
#         readline-dev \
#         sqlite-dev \
#         tcl-dev \
#         tk \
#         tk-dev \
#         util-linux-dev \
#         xz-dev \
#         zlib-dev \
#     ; \

RUN set -eux; \
    # wget https://www.python.org/ftp/python/${PYVERSION}/Python-${PYVERSION}.tgz; \
    # tar xf Python-${PYVERSION}.tgz; \
    # cd Python-${PYVERSION}; \
    # ./configure --with-optimizations --with-system-expat --with-system-ffi --without-ensurepip; \
    # make; \
    # make install; \
    # wget https://bootstrap.pypa.io/get-pip.py; \
    # python get-pip.py; \
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
    # sed -i /usr/lib/x86_64-linux-gnu/pkgconfig/python-3.7.pc -e 's/Libs: -lpython3.7m/Libs: -l:libpython3.7m.a/'; \
    ln -s /usr/lib/x86_64-linux-gnu/pkgconfig/python-3.7.pc /usr/lib/x86_64-linux-gnu/pkgconfig/python3.pc; \
    PKG_CONFIG_PATH=/usr/lib/x86_64-linux-gnu/pkgconfig go build --ldflags '-s -w' # --linkmode external --extldflags="-lm -lz -lexpat"'

CMD /bin/sh