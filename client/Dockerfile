FROM golang:1.14-alpine as build-env
ENV GO111MODULE=on

RUN mkdir /go/src/client
WORKDIR /go/src/client

COPY go.mod .
COPY go.sum .
RUN apk update && apk add git && apk add gcc && apk add bash && apk add libc-dev

RUN go mod download

COPY scripts/genClients.sh .
COPY scripts/run.sh .
COPY client .
