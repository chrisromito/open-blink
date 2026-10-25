# Overview

## Setup

1. Create the postgres_data directory
    ```shell
    mkdir postgres_data
    ```
1. Build the base torch image (makes subsequent runs MUCH faster):
    ```shell
    cd detection_service
    docker build -f torch.Dockerfile -t torch-base:1.0 .
    ```
1. Build the base images
    ```shell
    docker compose build runtime-base go-base
    ```
1. Run the "local"/testing workflow first:
     ```shell
     docker compose --file docker-compose.local.yaml up --build
     ```
1. Open your browser: http://localhost:4000
1. To run everything
   ```shell
   docker compose up --build
   ```

### MQTT Topics

- /start-stream/<device_id>: Allow users to turn on video feeds remotely
- /motion-detected/: Device notifies server of motion detection

#### Topic flow

| Topic                                             | Reads       | Writes      | Triggers                                          |
|---------------------------------------------------|-------------|-------------|---------------------------------------------------|
| open-blink/motion-detected/${DeviceID}            | Device      | DeviceImage | open-blink/devices/${DeviceID}/images/created     |
| open-blink/devices/${DeviceID}/images/created     | DeviceImage | Detection   | open-blink/devices/${DeviceID}/detections/created |
| open-blink/devices/${DeviceID}/images/updated     | ---         | ---         |                                                   |
| open-blink/devices/${DeviceID}/detections/created | Detection   | Event       | open-blink/devices/${DeviceID}/event/created      |
| open-blink/devices/${DeviceID}/event/created      | Event       | ---         |                                                   |
| open-blink/devices/${DeviceID}/event/updated      | Event       | ---         |                                                   |

## Helpful Commands

### Re-generate models.go

```shell
sqlc generate
```

### Lint devicecapture

```bash
cd devicecapture
golangci-lint run --fix
```

### Format devicecapture

```bash
cd devicecapture
golangci-lint fmt
```

### Generate Migrations

```shell
docker run --rm -v ./internal/postgres/migrations:/migrations \
  -e GOOSE_DRIVER="postgres" \
  -e GOOSE_COMMAND="create" \
  -e GOOSE_COMMAND_ARG="create_table_device_event sql" \
  -e GOOSE_DBSTRING=postgres://postgres:postgres@postgres:5432/openblink \
  ghcr.io/kukymbr/goose-docker:3.27.0
```
