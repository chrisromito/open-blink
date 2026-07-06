---------------------------------
-- Detection API
---------------------------------
-- name: GetRecentLabels :many
SELECT DISTINCT(label)
FROM detections
WHERE created_at > (NOW() - INTERVAL '7 days');

-- name: GetDetectionImagesByLabel :many
SELECT detections.id,
       detections.created_at,
       detections.label,
       detections.confidence,
       detections.bbox,
       detections.device_id,
       device_images.image_path,
       device_images.annotated_path
FROM device_images
         JOIN detections ON device_images.id = detections.image_id
WHERE label ILIKE ANY (@label::text[])
  AND (
    CASE
        WHEN @device_id::bigint = 0
            THEN detections.device_id IS NOT NULL
        ELSE detections.device_id = @device_id
        END
    )
  AND (
    CASE
        WHEN sqlc.narg('created_at')::timestamp with time zone IS NOT NULL
            THEN detections.created_at >= sqlc.narg('created_at')
        ELSE detections.created_at >= (NOW() - INTERVAL '7 days')
        END
    )
ORDER BY detections.created_at DESC
LIMIT 500;


---------------------------------
-- Timeline API
---------------------------------
-- name: GetDetectionTimeline :many
SELECT device_images.id::bigint as id,
       detections.image_id::bigint as image_id,
       device_images.image_path,
       device_images.annotated_path,
       device_images.created_at,
       device_images.device_id::bigint as device_id,
       detections.label,
       detections.confidence,
       detections.id::bigint as detection_id
FROM device_images
         RIGHT OUTER JOIN detections ON detections.image_id = device_images.id
WHERE (
          CASE
              WHEN $1::bigint = 0
                  THEN detections.device_id IS NOT NULL
              ELSE detections.device_id = $1
              END
          )
ORDER BY device_images.created_at DESC
LIMIT $2 OFFSET $3;


-- name: GetDetectionCount :one
SELECT COUNT(device_images.id)
FROM device_images
WHERE (
          CASE
              WHEN $1::bigint = 0
                  THEN device_images.device_id IS NOT NULL
              ELSE device_images.device_id = $1
              END
          );