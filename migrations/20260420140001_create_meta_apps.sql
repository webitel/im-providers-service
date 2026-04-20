-- +goose Up
-- Create table for Meta (Facebook/WhatsApp) application configurations
CREATE TABLE IF NOT EXISTS im_provider.meta_apps (
    id           UUID DEFAULT uuidv7() PRIMARY KEY,
    name         TEXT NOT NULL,
    app_id       TEXT NOT NULL UNIQUE,
    app_secret   TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    scopes       TEXT[] NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trigger to automatically update the updated_at timestamp
CREATE TRIGGER tr_meta_apps_updated 
BEFORE UPDATE ON im_provider.meta_apps 
FOR EACH ROW EXECUTE FUNCTION im_provider.update_timestamp();

-- +goose Down
DROP TABLE IF EXISTS im_provider.meta_apps;