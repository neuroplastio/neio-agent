ARG ARCH=

FROM ${ARCH}golang:1.22 AS build

RUN apt-get update \
  && apt-get install -y \
     libudev-dev \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

ADD go.mod go.sum ./
RUN go mod download

ADD . ./

RUN go build -o /neio-agent ./cmd/neio-agent

FROM ${ARCH}debian:bookworm-slim

COPY --from=build /neio-agent /neio-agent

ENTRYPOINT ["/neio-agent"]
