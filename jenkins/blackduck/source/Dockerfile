# Build agent for the Black Duck source scan.
#
# Detect needs a JRE and the Go toolchain. The Jenkinsfile builds this image
# with --pull, so every run picks up the latest golang:latest and the newest
# Temurin 21 JRE without anyone bumping versions by hand.

FROM golang:latest AS go

FROM eclipse-temurin:21-jre

LABEL maintainer="ITRS DevOps <devops@itrsgroup.com>"

RUN apt-get update -qq && \
    apt-get install -y -qq --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go /usr/local/go /usr/local/go

ENV PATH="/usr/local/go/bin:${PATH}" \
    GOTOOLCHAIN=local
