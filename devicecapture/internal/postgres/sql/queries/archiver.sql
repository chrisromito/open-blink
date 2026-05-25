------------------------------
-- Image/Asset archive queries
---------------------------------
-- name: GetArchiveTargets :many
SELECT *
    FROM device_images
WHERE created_at >= (NOW() - INTERVAL '7 days')
    ORDER BY created_at;