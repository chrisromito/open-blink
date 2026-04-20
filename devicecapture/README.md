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

