# Device service
####################

TLDR: Device = ESP32CAM that acts as a "server". This module is the client.
Devices update this system via MQTT
This system receives MJPEG streams directly from devices on the network.
Devices are configured in Postgres "open_blink" DB; "devices" table


### Server <-> MQTT <-> Device
Device management uses MQTT to push messages to devices
- /heartbeat/: Device heartbeat
- /start-stream/: Allow users to turn on video feeds remotely
- /motion-detected/: Device notifies server of motion detection
- /image/{DEVICE_ID}: Image payloads (as bytes)
- object-detection/{DEVICE_ID}

## Helpful Commands
### Re-generate models.go
```shell
sqlc generate
```

### Generate Migrations
```shell
docker run --rm -v ./internal/postgres/migrations:/migrations \
  -e GOOSE_COMMAND="create" \
  -e GOOSE_COMMAND_ARG="my_new_migration_name sql" \
  ghcr.io/kukymbr/goose-docker:v3.27.0
```