# Dockerfile to build cordial components and tar.gz/zip files
#
# docker build --tag cordial:$(cat VERSION) .
# docker create --name cordial-$(cat VERSION) cordial:$(cat VERSION)
# docker cp cordial-$(cat VERSION):/cordial-$(cat VERSION).tar.gz .
# docker cp cordial-$(cat VERSION):/cordial-$(cat VERSION).zip .
# docker rm cordial-$(cat VERSION)
#
# This builds binaries and libemail.so to run on older systems like Centos 7
#

FROM golang AS build
LABEL stage=cordial-build
# RUN mkdir /app
# WORKDIR /app
# RUN git clone https://github.com/ITRS-Group/cordial.git .
#
# or

# The "clean" lines below are in case of running this in a working
# directory with existing builds from outside the container, which may
# be from a different arch or environment
COPY ./ /app/cordial
WORKDIR /app/cordial/tools/geneos
RUN go mod tidy
RUN go clean
# RUN for i in amd64 arm64 arm; do mkdir bin.linux-$i; go build -o bin.linux-$i/geneos; done 
RUN go build
# RUN mkdir bin.windows; GOOS=windows go build -o bin.windows/geneos.exe
RUN GOOS=windows go build
WORKDIR /app/cordial/integrations/servicenow
RUN go mod tidy
RUN go clean
RUN go build
WORKDIR /app/cordial/integrations/pagerduty
RUN go mod tidy
RUN go clean
RUN go build
WORKDIR /app/cordial/libraries/libemail
RUN make

FROM debian AS cordial-build
LABEL stage=cordial-build
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
RUN apt update && apt install -y fontconfig zip
WORKDIR /app/cordial
COPY --from=build /app/cordial/VERSION /
COPY --from=build /app/cordial/tools/geneos/geneos /cordial/bin/
COPY --from=build /app/cordial/tools/geneos/geneos.exe /cordial/bin/
COPY --from=build /app/cordial/integrations/servicenow/servicenow /app/cordial/integrations/servicenow/ticket.sh /app/cordial/integrations/pagerduty/pagerduty /cordial/bin/
COPY --from=build /app/cordial/libraries/libemail/libemail.so /cordial/lib/
COPY --from=build /app/cordial/integrations/servicenow/servicenow.example.yaml /app/cordial/integrations/pagerduty/cmd/pagerduty.defaults.yaml /cordial/etc/geneos/
RUN mv /cordial /cordial-$(cat /VERSION)
WORKDIR /
RUN tar czf /cordial-$(cat /VERSION).tar.gz cordial-$(cat /VERSION) && zip -q -r /cordial-$(cat /VERSION).zip cordial-$(cat /VERSION) && rm -r /cordial-$(cat /VERSION)
CMD [ "bash" ]

FROM debian AS cordial-run
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
RUN apt update && apt install -y fontconfig
COPY --from=build /app/cordial/tools/geneos/geneos /bin/
RUN useradd -ms /bin/bash geneos
WORKDIR /home/geneos
USER geneos
CMD [ "bash" ]
