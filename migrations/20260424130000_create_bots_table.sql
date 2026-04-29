-- +goose Up
-- +goose StatementBegin

-- Create the bots table to store core identity information
-- This table acts as a central registry for all provider types
CREATE TABLE IF NOT EXISTS im_provider.bots (
    id         UUID DEFAULT uuidv7() PRIMARY KEY,
    sub        TEXT NOT NULL UNIQUE, 
    iss        TEXT NOT NULL,       
    gate_id    UUID NOT NULL REFERENCES im_provider.gates(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS im_provider.bots CASCADE;

-- +goose StatementEnd