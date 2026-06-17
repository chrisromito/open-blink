# Overview

## Setup
1. Build the base torch image (makes subsequent runs MUCH faster):
    ```shell
    cd detection_service
    docker build -f torch.Dockerfile -t torch-base:1.0 .
    ```
2. Run the "local"/testing workflow first:
    ```shell
    docker compose --file docker-compose.local.yaml up --build
    ```

3. Open your browser: http://localhost:4000

## Installation and Getting Started

This project is designed to run as a multi-service Docker application. The recommended local workflow is to use Docker Compose, which starts the MQTT broker, device capture services, mock device, detection service, and supporting infrastructure together.

### System Requirements

Install the following before running the project:

- Docker Desktop
- Docker Compose v2

### Docker Compose Files

The repository includes multiple Compose and Docker build files for different workflows.

| File | Purpose                                                                                  |
|---|------------------------------------------------------------------------------------------|
| `docker-compose.local.yaml` | Local development stack. Use this for normal development and testing.                    |
| `docker-compose.yaml` | Main Compose stack, typically used for the default or deployment-oriented configuration. |
| `mqtt.Dockerfile` | MQTT broker image/custom broker setup.                                                   |
| `devicecapture/docker/Dockerfile` | Main Go device capture image.                                                            |
| `devicecapture/docker/deviceserver.Dockerfile` | HTTP/device server image.                                                                |
| `devicecapture/docker/mockdevice.Dockerfile` | Mock MJPEG camera/device image.                                                          |
| `devicecapture/docker/publisher.Dockerfile` | MQTT publisher utility image.                                                            |
| `devicecapture/docker/test.Dockerfile` | Go test image.                                                                           |
| `detection_service/Dockerfile` | Detection service image.                                                                 |
| `detection_service/torch.Dockerfile` | Detection service base image with Torch/runtime dependencies.                            |

### Environment Configuration

Runtime configuration is provided through environment variables, usually from the Compose file or an `.env` file. See `.env.example`

Common MQTT-related variables include:

| Variable | Description |
|---|---|
| `MQTT_HOST` | MQTT broker URL/host used by services. |
| `MQTT_USER` | MQTT username, if authentication is enabled. |
| `MQTT_PASSWORD` | MQTT password, if authentication is enabled. |

### Start the Local Stack
You must build the torch base image first. This makes subsequent builds/deploys SIGNIFICANTLY faster (~2 hours -> 10 minutes on a RPi CM5 w/ 4GB RAM).
```bash
cd detection_service
docker build -f torch.Dockerfile -t torch-base:1.0 .
cd ..
```

From the repository root, start the local development environment:
```bash
docker compose -f docker-compose.local.yaml up --build
```


## MQTT Topics

The application uses MQTT to coordinate stream capture, image processing, and detection events.

| Topic | Producer | Consumer | Purpose |
|---|---|---|---|
| `motion-detected/#` | Motion/event source | `devicecapture` | Triggers capture of device streams. |
| `image/{deviceID}` | Go capture service | Detection service | Publishes captured image/frame metadata for a device. |
| `detections/{deviceID}` | Go camera/capture service | HTTP/WebSocket server | Publishes detection batches for a device. |
| `detections/#` | Go camera/capture service | WebSocket detection stream | Wildcard subscription used to proxy detection events to clients. |
| `detection/{deviceID}` | Python detection service | MQTT consumers | Publishes detection results for a specific device. |
| `end-stream/{deviceID}` | Go MQTT receiver/capture flow | MQTT consumers | Signals that a capture stream/session has ended. |

### Topic Details

- `{deviceID}` represents the target device identifier.
- `#` is the MQTT multi-level wildcard.
- `image/{deviceID}` is used to publish captured frames or frame metadata.
- `detections/{deviceID}` is used for detection batches published by the Go capture service.
- `detection/{deviceID}` is used by the Python detection service for per-device detection messages.
- `end-stream/{deviceID}` indicates that a capture session has completed.


## System Overview
The Go services connect to MQTT using the configured `MQTT_HOST`, `MQTT_USER`, and `MQTT_PASSWORD` values.

### Mock Device Workflow

The mock device service provides a local MJPEG camera-like stream for development. It exposes:

| Endpoint | Purpose |
|---|---|
| `/ping` | Health check endpoint. |
| `/stream` | MJPEG stream endpoint. |

When the mock device container is running, the capture service can consume this stream the same way it would consume a real device stream.

### Device Capture Workflow

The device capture service listens for MQTT messages that indicate when a stream should be captured. When a motion event is received, it captures frames from the configured device stream and publishes image/frame data to MQTT.

Typical flow:

1. A motion event is published to MQTT.
2. The capture service receives the event.
3. The capture service connects to the device stream.
4. Captured frames are stored and/or published.
5. Detection results are published back to MQTT.
6. HTTP/WebSocket services proxy image streams and detection events to clients.

### Detection Service Workflow

The detection service consumes image/frame messages from MQTT, performs object detection, and publishes detection results back to MQTT.

Typical flow:

1. Subscribe to image topics.
2. Receive captured frame metadata or image payloads.
3. Run detection.
4. Publish detection messages to MQTT.
5. Downstream services consume detection events.
