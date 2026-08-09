-- Devices (Cameras)
CREATE TABLE devices
(
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       varchar(250) NOT NULL,
    device_url varchar(250) NOT NULL,
    UNIQUE (name)
);

-- Heartbeats
CREATE TABLE device_heartbeats
(
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id  bigint                                 NOT NULL
        CONSTRAINT device_heartbeats_device__fk
            REFERENCES devices
            ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT NOW() NOT NULL
);

CREATE INDEX device_heartbeats__created_at__index
    ON device_heartbeats (created_at);

CREATE INDEX device_heartbeats__device_id__idx
    ON device_heartbeats (device_id);

-- Images
CREATE TABLE device_images
(
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id      bigint                                 NOT NULL
        CONSTRAINT device_images_device__fk
            REFERENCES devices
            ON DELETE CASCADE,
    event_id       bigint
        CONSTRAINT device_images_event__fk
            REFERENCES detection_events
            ON DELETE CASCADE,
    image_path     varchar(250)                           NOT NULL,
    annotated_path varchar(250),
    created_at     timestamp with time zone DEFAULT NOW() NOT NULL,
    UNIQUE (image_path),
    UNIQUE (annotated_path)
);

CREATE INDEX device_images__created_at_idx
    ON device_images (created_at);

-- Detections
CREATE TABLE detections
(
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id  bigint       NOT NULL
        CONSTRAINT detections_device__fk
            REFERENCES devices
            ON DELETE CASCADE,
    image_id   bigint
        CONSTRAINT detections_image__fk
            REFERENCES device_images
            ON DELETE CASCADE,
    event_id   bigint
        CONSTRAINT detections_event__fk
            REFERENCES detection_events
            ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT NOW() NOT NULL,
    label      varchar(250) NOT NULL,
    confidence float        NOT NULL    DEFAULT 0.0,
    bbox       float[][2]
);

CREATE INDEX detections__created_at__index
    ON detections (created_at);

CREATE INDEX detections__device_id__idx
    ON detections (device_id);


-- Events
CREATE TABLE detection_events
(
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id  bigint NOT NULL
        CONSTRAINT detection_event_device__fk
            REFERENCES devices
            ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT NOW() NOT NULL,
    ended_at   timestamp with time zone,
    labels     varchar(250),
    state      int    NOT NULL          DEFAULT 1
);

CREATE INDEX detection_events__created_at_idx
    ON detection_events (created_at);