-- Devices (Cameras)
CREATE TABLE devices
(
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       varchar(250) NOT NULL,
    device_url varchar(250) NOT NULL
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
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id  bigint                                 NOT NULL
        CONSTRAINT device_images_device__fk
            REFERENCES devices
            ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT NOW() NOT NULL,
    image_path varchar(250)                           NOT NULL,
    UNIQUE (image_path)
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
    created_at timestamp with time zone DEFAULT NOW() NOT NULL,
    label      varchar(250) NOT NULL,
    confidence float        NOT NULL    DEFAULT 0.0
);

CREATE INDEX detections__created_at__index
    ON detections (created_at);

CREATE INDEX detections__device_id__idx
    ON detections (device_id);


