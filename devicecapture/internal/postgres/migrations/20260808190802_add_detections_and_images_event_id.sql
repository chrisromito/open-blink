-- +goose Up
ALTER TABLE device_images
    ADD COLUMN event_id bigint,
    ADD CONSTRAINT device_images_event__fk
        FOREIGN KEY (event_id)
            REFERENCES detection_events
            ON DELETE CASCADE;

ALTER TABLE detections
    ADD COLUMN event_id bigint,
    ADD CONSTRAINT detections_event__fk
        FOREIGN KEY (event_id)
            REFERENCES detection_events
            ON DELETE CASCADE;
-- +goose Down
ALTER TABLE device_images
    DROP CONSTRAINT detections_event__fk,
    DROP COLUMN event_id;

ALTER TABLE detections
    ADD COLUMN event_id bigint,
    ADD CONSTRAINT detections_event__fk
        FOREIGN KEY (event_id)
            REFERENCES detection_events
            ON DELETE CASCADE;