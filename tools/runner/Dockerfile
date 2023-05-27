FROM golang:alpine AS easy-novnc
RUN apk update
RUN apk add build-base
# RUN apk add tigervnc
COPY easy-novnc/go.* easy-novnc/server.go easy-novnc/index.html easy-novnc/index_generate.go easy-novnc/novnc_generate.go /easy-novnc/
WORKDIR /easy-novnc
RUN go generate
RUN go build --ldflags '-linkmode external -extldflags=-static'

ARG TIGERVNC_BASE=https://sourceforge.net/projects/tigervnc/files/stable
ARG TIGERVNC_VERSION=1.13.1
ARG TIGERVNC_DISTRO=x86_64
ARG TIGERVNC_HOME=/

RUN wget ${TIGERVNC_BASE}/${TIGERVNC_VERSION}/tigervnc-${TIGERVNC_VERSION}.${TIGERVNC_DISTRO}.tar.gz -P /tmp \
    &&  tar xzf /tmp/tigervnc-${TIGERVNC_VERSION}.${TIGERVNC_DISTRO}.tar.gz --strip 1 -C ${TIGERVNC_HOME} tigervnc-${TIGERVNC_VERSION}.${TIGERVNC_DISTRO}/usr/bin/Xvnc

FROM ubuntu:latest
ARG DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC
RUN apt update
RUN apt upgrade -y --no-install-recommends
RUN apt install -y --no-install-recommends ca-certificates
# RUN apt install -y --no-install-recommends wget
RUN apt install -y --no-install-recommends dbus-x11 xauth xinit xfonts-base fonts-freefont-ttf libxtst6
RUN apt install -y --no-install-recommends xfce4 xfce4-terminal elementary-xfce-icon-theme
RUN apt install -y --no-install-recommends xdg-user-dirs
# RUN apt install -y --no-install-recommends xfce4-power-manager xfce4-notifyd xfce4-cpugraph-plugin
RUN apt install -y --no-install-recommends libgl1

ADD https://github.com/ITRS-Group/cordial/releases/latest/download/geneos /usr/bin/
RUN chmod +x /usr/bin/geneos

RUN useradd -m -s /bin/bash geneos

RUN useradd -m -s /bin/bash xserver
COPY test.* /home/xserver/
RUN chown xserver /home/xserver/test.*

RUN useradd -m -s /bin/bash novnc
COPY test.* /home/novnc/
RUN chown novnc /home/novnc/test.*

COPY --from=easy-novnc /easy-novnc/easy-novnc /usr/bin/easy-novnc
COPY --from=easy-novnc /usr/bin/Xvnc /usr/bin/Xvnc
COPY ./runner /root/

RUN mkdir /tmp/.font-unix
RUN mkdir /tmp/.X11-unix
RUN mkdir /tmp/.ICE-unix
RUN chmod 1777 /tmp/.font-unix /tmp/.X11-unix /tmp/.ICE-unix

RUN echo DESKTOP=.Desktop > /etc/xdg/user-dirs.defaults

VOLUME ${PWD}:/opt:ro
EXPOSE 6901

ENTRYPOINT [ "/root/runner" ]