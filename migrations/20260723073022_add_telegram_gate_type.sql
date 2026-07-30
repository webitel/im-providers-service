-- +goose Up
-- +goose StatementBegin
CREATE TABLE im_provider.telegram(
  gate_id UUID PRIMARY KEY NOT NULL REFERENCES im_provider.gates(id) ON DELETE CASCADE,
  token TEXT NOT NULL,
  uri TEXT NOT NULL UNIQUE,
  webhook_secret TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE im_provider.telegram;

-- +goose StatementEnd
