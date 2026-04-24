-- +goose Up
-- Create specialized table for Facebook-specific gateway settings
CREATE TABLE IF NOT EXISTS im_provider.gate_facebook (
    gate_id      UUID PRIMARY KEY REFERENCES im_provider.gates(id) ON DELETE CASCADE,
    contact_id   UUID NOT NULL, -- Target contact or system recipient ID
    meta_app_id  UUID NOT NULL REFERENCES im_provider.meta_apps(id),
    page_id      TEXT NOT NULL,
    page_token   TEXT NOT NULL,
    UNIQUE (meta_app_id, page_id)
);

-- +goose Down
DROP TABLE IF EXISTS im_provider.gate_facebook CASCADE;