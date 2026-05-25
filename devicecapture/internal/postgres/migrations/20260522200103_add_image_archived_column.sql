-- +goose Up
ALTER TABLE device_images
    ADD COLUMN archived boolean DEFAULT FALSE NOT NULL;

-- +goose Down
ALTER TABLE device_images
    DROP COLUMN archived;
