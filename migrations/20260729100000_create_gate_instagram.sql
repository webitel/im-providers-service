-- +goose Up
-- Create specialized table for Instagram-specific gateway settings
CREATE TABLE IF NOT EXISTS im_provider.instagram (
    gate_id             UUID PRIMARY KEY REFERENCES im_provider.gates(id) ON DELETE CASCADE,
    meta_app_id         UUID NOT NULL REFERENCES im_provider.meta_apps(id),
    business_account_id TEXT NOT NULL,
    ig_access_token     TEXT NOT NULL,
    name                TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (meta_app_id, business_account_id)
);

-- Trigger to automatically update the updated_at timestamp
CREATE TRIGGER tr_instagram_updated
BEFORE UPDATE ON im_provider.instagram
FOR EACH ROW EXECUTE FUNCTION im_provider.update_timestamp();

-- +goose Down
DROP TABLE IF EXISTS im_provider.instagram CASCADE;
