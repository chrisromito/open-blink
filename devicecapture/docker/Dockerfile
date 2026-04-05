FROM debian:bookworm-slim AS base
LABEL authors="chris"

WORKDIR /usr/src/app

FROM golang:1.24.3 AS build
WORKDIR /usr/src/build

COPY ../go.mod go.sum ./
RUN go mod download

COPY . .
COPY ./devices.json ./bin/devices.json
RUN go build -v -o ./bin/devicecapture ./cmd/devicecapture

FROM base AS final
WORKDIR /usr/src/app
COPY --from=build /usr/src/build/bin .

RUN mkdir -p videos

CMD ["./devicecapture"]