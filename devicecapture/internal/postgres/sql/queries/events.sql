-- name: GetEvents :many
SELECT *
FROM detection_events
WHERE (
    CASE
        -- 1 = device_id
        WHEN @device_id::bigint = 0
            THEN detection_events.device_id IS NOT NULL
        ELSE detection_events.device_id = @device_id
        END
    )
  AND (
    -- $2 = state
    CASE
        WHEN @state::int = 0
            THEN detection_events.state IS NOT NULL
        ELSE detection_events.state = @state
        END
    )
ORDER BY created_at DESC
-- $3, $4 = limit, offset
LIMIT @lim OFFSET @off;

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
SET labels =
        CASE
            WHEN @set_labels::bool
                THEN @labels::text
            ELSE labels
            END,
    state  =
        CASE
            WHEN @set_state::bool
                THEN @state::INT
            ELSE state
            END
WHERE id = @id
RETURNING *;