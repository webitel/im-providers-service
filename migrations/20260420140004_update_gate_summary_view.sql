-- +goose Up
-- +goose StatementBegin

-- Drop the old version if it exists and create a clean view without hardcoded URLs.
-- The webhook logic is now completely handled by the application layer.
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
LEFT JOIN im_provider.facebook fb ON g.id = fb.gate_id
LEFT JOIN im_provider.meta_apps ma ON fb.meta_app_id = ma.id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Completely remove the view on rollback.
DROP VIEW IF EXISTS im_provider.gate_summary;

-- +goose StatementEnd