------------------------------
-- Image/Asset archive queries
---------------------------------
-- name: GetArchiveTargets :many
SELECT *
FROM device_images
WHERE archived = FALSE
  AND created_at >= (NOW() - INTERVAL '7 days')
ORDER BY created_at
LIMIT 500;

-- name: SetArchived :exec
UPDATE device_images
SET image_path = @image_path,
    archived   = TRUE
WHERE id = @id;

-- name: PurgeOldestTargets :exec
DELETE
FROM device_images
WHERE id IN (SELECT id
             FROM device_images
             WHERE archived = FALSE
               AND created_at >= (NOW() - INTERVAL '7 days')
             ORDER BY created_at
             LIMIT 500);
