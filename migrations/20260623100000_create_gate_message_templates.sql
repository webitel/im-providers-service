-- +goose Up
-- gate_message_templates stores per-gate overrides for system event messages.
-- When a row exists for (gate_id, event_type) the template takes precedence over
-- the built-in default defined in TemplateRenderer.
-- Template values are Go text/template strings that may reference vars passed
-- by im-thread-service (e.g. {{.new_member_role}}).
CREATE TABLE IF NOT EXISTS im_provider.gate_message_templates (
    gate_id    UUID        NOT NULL REFERENCES im_provider.gates(id) ON DELETE CASCADE,
    event_type TEXT        NOT NULL,
    template   TEXT        NOT NULL,
    domain_id  BIGINT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (gate_id, event_type)
);

-- +goose Down
DROP TABLE IF EXISTS im_provider.gate_message_templates;
