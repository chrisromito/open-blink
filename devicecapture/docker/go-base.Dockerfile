# docker/go-base.Dockerfile
FROM golang:1.26.0

WORKDIR /usr/src/build

COPY go.mod go.sum ./
RUN go mod download

RUN go install gotest.tools/gotestsum@latest