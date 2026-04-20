-- +goose Up
-- +goose StatementBegin
-- Create schema for IM providers service
CREATE SCHEMA IF NOT EXISTS im_provider;

-- Function to automatically update the updated_at column on row changes
CREATE OR REPLACE FUNCTION im_provider.update_timestamp() RETURNS TRIGGER AS $$
BEGIN 
    NEW.updated_at = NOW(); 
    RETURN NEW; 
END; $$ language 'plpgsql';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Use CASCADE to remove the function and all triggers depending on it
DROP FUNCTION IF EXISTS im_provider.update_timestamp() CASCADE;

-- Use CASCADE to drop the schema and all internal objects (tables, views, etc.)
DROP SCHEMA IF EXISTS im_provider CASCADE;
-- +goose StatementEnd