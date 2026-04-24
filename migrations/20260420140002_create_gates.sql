-- +goose Up
-- Create base table for all communication gateways (channels)
CREATE TABLE IF NOT EXISTS im_provider.gates (
    id          UUID DEFAULT uuidv7() PRIMARY KEY,
    dc          BIGINT NOT NULL, -- domain identifier
    name        TEXT NOT NULL,
    type        TEXT NOT NULL, -- e.g., 'facebook', 'whatsapp', 'telegram'
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trigger to automatically update the updated_at timestamp
CREATE TRIGGER tr_gates_updated 
BEFORE UPDATE ON im_provider.gates 
FOR EACH ROW EXECUTE FUNCTION im_provider.update_timestamp();

-- +goose Down
DROP TABLE IF EXISTS im_provider.gates;