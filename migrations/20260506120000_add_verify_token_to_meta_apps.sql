-- +goose Up
ALTER TABLE im_provider.meta_apps
    ADD COLUMN verify_token TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE im_provider.meta_apps
    DROP COLUMN verify_token;
