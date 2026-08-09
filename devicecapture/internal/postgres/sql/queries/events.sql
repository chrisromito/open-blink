-- name: GetEvent :one
SELECT * FROM detection_events WHERE id = @id LIMIT 1;

-- name: GetEvents :many
SELECT *
FROM detection_events
WHERE (
    CASE
        WHEN @device_id::bigint = 0
            THEN detection_events.device_id IS NOT NULL
        ELSE detection_events.device_id = @device_id
        END
    )
  AND (
    CASE
        WHEN @state::int = 0
            THEN detection_events.state IS NOT NULL
        ELSE detection_events.state = @state
        END
    )
  AND (
    CASE
        WHEN @startdt::timestamp = '0001-01-01 00:00:00.000000 +00:00'
            THEN detection_events.created_at IS NOT NULL
        ELSE detection_events.created_at >= @startdt
        END
    )
  AND (
    CASE
        WHEN @enddt::timestamp = '0001-01-01 00:00:00.000000 +00:00'
            THEN detection_events.created_at IS NOT NULL
        ELSE detection_events.created_at <= @enddt
        END
    )
ORDER BY created_at DESC
LIMIT @lim OFFSET @OFF;

-- name: StartEvent :one
INSERT INTO detection_events(id, device_id, created_at, ended_at, labels, state)
VALUES (DEFAULT, @device_id, DEFAULT, @ended_at, @labels, @state)
RETURNING *;


-- name: EndEvent :one
UPDATE detection_events
SET state    = 3,
    ended_at = NOW()
WHERE id = @id
RETURNING *;

-- name: UpdateEvent :one
UPDATE detection_events
SET labels     =
        CASE
            WHEN @set_labels::bool
                THEN @labels::text
            ELSE labels
            END,
    state      =
        CASE
            WHEN @set_state::bool
                THEN @state::INT
            ELSE state
            END,
    created_at =
        CASE
            WHEN @set_created_at::bool
                THEN @created_at::timestamp
            ELSE created_at
            END,
    ended_at   =
        CASE
            WHEN @set_ended_at::bool
                THEN @ended_at::timestamp
            ELSE ended_at
            END
WHERE id = @id
RETURNING *;


-- name: GetEventDetails :many
SELECT detection_events.*,
       device_images.id      AS image_id,
       device_images.annotated_path,
       device_images.image_path,
       detections.id         AS detection_id,
       detections.label,
       detections.confidence,
       detections.created_at AS detected_at,
       detections.bbox,
       detections.image_id   AS detected_image_id
FROM detection_events
         JOIN device_images ON device_images.event_id = detection_events.id
         JOIN detections ON detections.event_id = detection_events.id
WHERE detection_events.id = @id
  AND detections.created_at >= detection_events.created_at
  AND detections.created_at <= detection_events.ended_at
LIMIT 100;


-- name: EndStaleDetectionEvents :exec
UPDATE detection_events
SET ended_at = NOW(),
    state    = 3
WHERE ended_at = '0001-01-01 00:00:00.000000 +00:00';