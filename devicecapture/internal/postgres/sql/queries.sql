-- name: GetDevices :many
SELECT *
FROM devices
ORDER BY name;

-- name: GetDeviceById :one
SELECT *
FROM devices
WHERE id = $1
LIMIT 1;

-- name: CreateTestDevice :one
INSERT INTO devices (id, name, device_url)
VALUES (DEFAULT, 'mockdevice', 'http://mock_device:8080')
RETURNING *;

-- name: DeleteTestDevices :exec
DELETE
FROM devices
WHERE name ILIKE '%mockdevice%';


-- name: DeleteDevice :exec
DELETE
FROM devices
WHERE id = $1;

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
  AND created_at >= $2
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
VALUES (DEFAULT, $1, NOW())
RETURNING *;


-- name: DeleteBeats :exec
DELETE
FROM device_heartbeats
WHERE device_id = $1;


-----------------
-- Detections
-----------------
-- name: CreateDetection :one
INSERT INTO detections (id, device_id, label, confidence, image_id)
VALUES (DEFAULT, $1, $2, $3, $4)
RETURNING *;

-- name: CreateDetections :copyfrom
INSERT INTO detections (device_id, label, confidence, image_id)
VALUES ($1, $2, $3, $4);

-- name: GetDetectionsAfter :many
SELECT *
FROM detections
WHERE created_at >= $1
ORDER BY created_at DESC;

-- name: GetDeviceDetectionsAfter :many
SELECT *
FROM detections
WHERE device_id = @device_id
  AND created_at >= @created_at
    AND image_id = COALESCE(@image_id, image_id)
ORDER BY created_at DESC;


-- name: DeleteDetections :exec
DELETE
FROM detections
WHERE device_id = $1;


------------ Images

-- name: CreateImage :one
INSERT INTO device_images (id, device_id, created_at, image_path)
VALUES (DEFAULT, @device_id, DEFAULT, @image_path)
RETURNING *;

-- name: GetDeviceImages :many
SELECT *
FROM device_images
WHERE device_id = @device_id;