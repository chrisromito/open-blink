# Overview

## Setup

See [open-blink README.md](../README.md)

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

#### FIXME

[x] - `Images` topic triggers Detection Handlers
[x] - `Detection` topics trigger Event Handlers
[ ] - `Motion Detected` topic triggers MotionCapture

```go
package receiver

type FrameMsg struct {
	DeviceId  string `json:"device_id"`
	FileName  string `json:"file_name"`
	Timestamp int64  `json:"timestamp"`
	Url       string `json:"url"`
}
```

- detections/<device_id>: Receive updates

```go
package receiver

import "time"

// DetectionsMsg shape of MQTT messages that get published to /detections/<device_id>
type DetectionsMsg struct {
	ID         int64         `db:"id" json:"id"`
	DeviceID   int64         `db:"device_id" json:"device_id"`
	CreatedAt  time.Time     `db:"created_at" json:"created_at"`
	Url        string        `json:"url"`
	Detections []QtDetection `db:"detections" json:"detections"`
}

// QtDetection shape of nested `detections` within a DetectionsMsg
type QtDetection struct {
	ID         int64       `db:"id" json:"id"`
	Label      string      `db:"label" json:"label"`
	Confidence float64     `db:"confidence" json:"confidence"`
	Bbox       [][]float64 `db:"bbox" json:"bbox"`
}
```

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
