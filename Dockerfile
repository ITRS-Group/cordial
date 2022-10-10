# this is purely for building convenience

FROM golang AS build
COPY ./ /app/cordial
WORKDIR /app/cordial/tools/geneos
RUN go mod tidy
RUN go clean
RUN go build
WORKDIR /app/cordial/integrations/servicenow/snow_client
RUN go mod tidy
RUN go clean
RUN go build
WORKDIR /app/cordial/integrations/servicenow/snow_router
RUN go mod tidy
RUN go clean
RUN go build
WORKDIR /app/cordial/libraries/libemail
RUN make clean
RUN make

FROM debian
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
RUN apt update && apt install -y fontconfig
COPY --from=build /app/cordial/tools/geneos/geneos /bin
#
RUN mkdir /cordial
VOLUME /cordial
COPY --from=build /app/cordial/tools/geneos/geneos /cordial
COPY --from=build /app/cordial/integrations/servicenow/snow_client/snow_client /cordial
COPY --from=build /app/cordial/integrations/servicenow/snow_router/snow_router /cordial
COPY --from=build /app/cordial/libraries/libemail/libemail.so /cordial
#
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
# COPY archives packages/downloads
# RUN chown -R geneos:geneos packages
USER geneos
CMD [ "bash" ]
