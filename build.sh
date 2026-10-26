#!/usr/bin/env bash
set -euo pipefail

REGISTRY="192.168.0.15:5000"
PLATFORM="linux/arm64"
COMPOSE="docker-compose.deploy.yaml"

# Build and push ARM64 base images
docker buildx build \
  --platform "$PLATFORM" \
  -f detection_service/torch.Dockerfile \
  -t "$REGISTRY/torch-base:1.0" \
  --push \
  detection_service

docker buildx build \
  --platform "$PLATFORM" \
  -f devicecapture/docker/runtime-base.Dockerfile \
  -t "$REGISTRY/openblink-runtime-base:local" \
  --push \
  devicecapture

docker buildx build \
  --platform "$PLATFORM" \
  -f devicecapture/docker/go-base.Dockerfile \
  -t "$REGISTRY/openblink-go-base:local" \
  --push \
  devicecapture

# Build and push application images
docker compose -f "$COMPOSE" build --push \
  blink_detection_service \
  devicecapture \
  deviceserver

echo "All ARM64 production images built and pushed."