# this is purely for manual testing

FROM golang AS build
COPY ./ /app/cordial
WORKDIR /app/cordial/tools/geneos
RUN go build

FROM debian
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
RUN apt update && apt install -y fontconfig
COPY --from=build /app/cordial/tools/geneos/geneos /bin
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
# COPY archives packages/downloads
# RUN chown -R geneos:geneos packages
USER geneos
CMD [ "bash" ]
