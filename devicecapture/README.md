# Overview

## Setup
1. Create the postgres_data directory
```shell
mkdir postgres_data
```

2. Build the base torch image (makes subsequent runs MUCH faster):
 ```shell
 cd detection_service
 docker build -f torch.Dockerfile -t torch-base:1.0 .
 ```
3.Run the "local"/testing workflow first:
 ```shell
 docker compose --file docker-compose.local.yaml up --build
 ```

4. Open your browser: http://localhost:4000


### MQTT Topics
- /start-stream/<device_id>: Allow users to turn on video feeds remotely
- /motion-detected/: Device notifies server of motion detection
- /image/<device_id>: Image payloads as JSON
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
