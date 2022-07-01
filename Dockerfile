# Usage:
#
#  docker build -t cmdns-cli .
#  docker run --rm -ti cmdns-cli -help

FROM golang:1.18-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /usr/bin/cmdns-cli

ENTRYPOINT [ "/usr/bin/cmdns-cli" ]
