-- +goose Up
CREATE TABLE detection_events
(
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id  bigint                                 NOT NULL
        CONSTRAINT detection_event_device__fk
            REFERENCES devices
            ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT NOW() NOT NULL,
    ended_at timestamp with time zone,
    labels      varchar(250),
    state      int not null default 1
);

CREATE INDEX detection_events__created_at_idx
    ON detection_events (created_at);

-- +goose Down
DROP TABLE IF EXISTS detection_events;
