FROM golang:1.17.1-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN apk add build-base

RUN go build -o eth-brute
