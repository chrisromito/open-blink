FROM debian:bookworm-slim AS base
LABEL authors="chris"

WORKDIR /usr/src/app

FROM golang:1.24.3 AS build
WORKDIR /usr/src/build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o ./bin/mockdevice ./cmd/mockdevice

FROM base AS final
WORKDIR /usr/src/app
COPY --from=build /usr/src/build/bin .

EXPOSE 8080
CMD ["./mockdevice"]