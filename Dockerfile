# Dockerfile to help build cordial components
#
# docker build --tag cordial .
# docker run -it cordial bash
# .. then extract /cordial-[VERSION].tar.gz and use that
#
# This allows binaries and libemail.so to run on older systems like Centos 7
#

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
COPY --from=build /app/cordial/VERSION /
COPY --from=build /app/cordial/tools/geneos/geneos /bin
COPY --from=build /app/cordial/tools/geneos/geneos /cordial/
COPY --from=build /app/cordial/integrations/servicenow/snow_client/snow_client /cordial/
COPY --from=build /app/cordial/integrations/servicenow/snow_router/snow_router /cordial/
COPY --from=build /app/cordial/libraries/libemail/libemail.so /cordial/
RUN mv /cordial /cordial-$(cat /VERSION)
WORKDIR /
RUN tar czf /cordial-$(cat /VERSION).tar.gz cordial-$(cat /VERSION)
RUN rm -r /cordial-$(cat /VERSION)
# and we get a test environment too
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
# local package files can be copied in here
# COPY archives packages/downloads
# RUN chown -R geneos:geneos packages
USER geneos
CMD [ "bash" ]
