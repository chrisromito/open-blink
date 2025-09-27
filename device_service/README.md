# Device service
####################

TLDR: Communicates with camera devices via MQTT & WebSockets

### Device -> Server
- WS: Stream video feeds (via MJPEG frames) when they detect motion
  - Video streams are saved to the local file system

### Server <-> MQTT <-> Device
TODO: Can devices use MQTT to register themselves?
Device management uses MQTT to push messages to devices
- /heartbeat/: Device heartbeat
- /start-stream/: Allow users to turn on video feeds remotely
- /motion-detected/: Device notifies server of motion detection

## Future
- Object detection via detection_service
- Push notifications?
