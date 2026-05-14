------------------------------
-- Detection API
---------------------------------
-- name: GetRecentLabels :many
SELECT DISTINCT(label)
FROM detections
WHERE created_at > NOW() - INTERVAL '7 days';

-- name: GetDetectionImagesByLabel: many
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
ORDER BY detections.created_at DESC;
