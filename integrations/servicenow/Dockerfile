# this is old, not used and kept for review later

FROM golang:1.14.6
WORKDIR /go/src/github.com/itrs-group/cordial/integrations/servicenow/starting_api/
COPY go.mod go.sum /go/src/github.com/itrs-group/cordial/integrations/servicenow
COPY ./snow_router /go/src/github.com/itrs-group/cordial/integrations/servicenow/snow_router
COPY ./servicenow /go/src/github.com/itrs-group/cordial/integrations/servicenow/servicenow
COPY ./settings /go/src/github.com/itrs-group/cordial/integrations/servicenow/settings
RUN cd ../snow_router && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o snow_router

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY /starting_api/env.yaml .
COPY --from=0 /go/src/github.com/itrs-group/cordial/integrations/servicenow/snow_router/snow_router .
CMD ["./snow_router"]