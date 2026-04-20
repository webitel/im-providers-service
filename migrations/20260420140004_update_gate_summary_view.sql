-- +goose Up
-- +goose StatementBegin

-- We must drop the view because CREATE OR REPLACE VIEW does not allow removing columns.
-- The webhook_url column is removed as it is now generated dynamically in the application layer.
DROP VIEW IF EXISTS im_provider.gate_summary;

CREATE VIEW im_provider.gate_summary AS
SELECT 
    g.id,
    g.name,
    g.type, 
    CASE WHEN g.enabled THEN 'active' ELSE 'disabled' END AS status,
    COALESCE(fb.page_id, 'N/A') AS contact,
    ma.id::text AS provider_app_id,
    g.created_at,
    g.updated_at
FROM im_provider.gates g
LEFT JOIN im_provider.gate_facebook fb ON g.id = fb.gate_id
LEFT JOIN im_provider.meta_apps ma ON fb.meta_app_id = ma.id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP VIEW IF EXISTS im_provider.gate_summary;

-- Recreate the original version of the view with the hardcoded webhook_url path.
CREATE VIEW im_provider.gate_summary AS
SELECT 
    g.id,
    g.name,
    g.type, 
    CASE WHEN g.enabled THEN 'active' ELSE 'disabled' END AS status,
    '/webhooks/' || g.type || '/' || g.id AS webhook_url,
    COALESCE(fb.page_id, 'N/A') AS contact,
    ma.id::text AS provider_app_id,
    g.created_at,
    g.updated_at
FROM im_provider.gates g
LEFT JOIN im_provider.gate_facebook fb ON g.id = fb.gate_id
LEFT JOIN im_provider.meta_apps ma ON fb.meta_app_id = ma.id;

-- +goose StatementEnd