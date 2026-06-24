-- +goose Up
ALTER TABLE device_images
    ADD COLUMN annotated_path varchar(250);

-- +goose Down
ALTER TABLE device_images
    DROP COLUMN annotated_path;
