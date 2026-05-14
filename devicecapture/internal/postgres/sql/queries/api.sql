------------------------------
-- Detection API
---------------------------------
-- name: GetRecentLabels :many
SELECT DISTINCT(label)
FROM detections
WHERE created_at > NOW() - INTERVAL '7 days';

-- name: GetDetectionImagesByLabel :many
SELECT detections.id,
       detections.created_at,
       detections.label,
       detections.confidence,
       detections.bbox,
       detections.device_id,
       device_images.image_path
FROM detections
         JOIN device_images ON device_images.id = detections.id
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
        ELSE detections.created_at >= NOW() - INTERVAL '7 days'
        END
    )
ORDER BY detections.created_at DESC;
