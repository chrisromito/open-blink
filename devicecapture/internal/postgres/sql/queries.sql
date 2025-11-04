-- name: GetDevices :many
SELECT *
FROM devices
order by name;

-- name: GetDeviceById :one
SELECT *
FROM devices
WHERE id = $1
LIMIT 1;

-- name: CreateTestDevice :one
INSERT INTO devices (id, name, device_url)
VALUES (DEFAULT, 'mockdevice', 'http://localhost:8080')
RETURNING *;


-- name: CreateDevice :one
INSERT INTO devices (id, name, device_url)
VALUES (DEFAULT, $1, $2)
RETURNING *;

-- name: UpdateDevice :exec
UPDATE devices
SET name       = $2,
    device_url = $3
WHERE id = $1;


-----------------
-- HeartBeats
-----------------

-- name: GetDeviceHeartBeats :many
SELECT *
FROM device_heartbeats
WHERE device_id = $1
  and created_at >= $2
ORDER BY created_at DESC;

-- name: HeartBeatsAfter :many
SELECT *
FROM device_heartbeats
WHERE created_at >= $1
ORDER BY created_at DESC;

-- name: LatestBeats :many
SELECT DISTINCT(device_heartbeats.device_id), device_heartbeats.created_at, devices.*
FROM device_heartbeats
         JOIN devices ON devices.id = device_heartbeats.device_id
ORDER BY device_heartbeats.created_at DESC;

-- name: RecordBeat :one
INSERT INTO device_heartbeats (id, device_id, created_at)
VALUES (DEFAULT, $1, now())
RETURNING *;


-- name: DeleteBeats :exec
DELETE
FROM device_heartbeats
WHERE device_id = $1;


-----------------
-- Detections
-----------------
-- name: CreateDetection :one
INSERT INTO detections (id, device_id, label, confidence)
VALUES (DEFAULT, $1, $2, $3)
RETURNING *;

-- name: GetDetectionsAfter :many
SELECT *
FROM detections
WHERE created_at >= $1
ORDER BY created_at DESC;

-- name: GetDeviceDetectionsAfter :many
SELECT *
FROM detections
WHERE device_id = $1
  AND created_at >= $2
ORDER BY created_at DESC;


-- name: DeleteDetections :exec
DELETE
FROM detections
WHERE device_id = $1;
