# this is purely for manual testing

FROM golang AS build
WORKDIR /app
COPY go.mod /app/
COPY go.sum /app/
COPY main.go /app/

COPY cmd /app/cmd
COPY pkg /app/pkg
COPY internal /app/internal

WORKDIR /app
RUN go build

FROM debian
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
RUN apt update && apt install -y fontconfig
COPY --from=build /app/geneos /bin
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
# COPY archives packages/downloads
# RUN chown -R geneos:geneos packages
USER geneos
CMD [ "bash" ]
