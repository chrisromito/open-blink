# docker/runtime-base.Dockerfile
FROM debian:bookworm-slim

LABEL authors="chris"

WORKDIR /usr/src/app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /usr/src/app/static /static/videos