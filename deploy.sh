#!/usr/bin/env bash
#!/usr/bin/env bash
set -euo pipefail

COMPOSE="docker-compose.deploy.yaml"

# Pull the published images
docker compose -f "$COMPOSE" pull

# Start/update the production stack without building locally
docker compose -f "$COMPOSE" up -d --no-build --remove-orphans

# Show service status
docker compose -f "$COMPOSE" ps