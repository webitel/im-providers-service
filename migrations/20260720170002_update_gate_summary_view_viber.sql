-- +goose Up
-- +goose StatementBegin

-- Extend the cross-provider summary view so Viber gates report a contact identity.
DROP VIEW IF EXISTS im_provider.gate_summary;

CREATE VIEW im_provider.gate_summary AS
SELECT
    g.id,
    g.name,
    g.type,
    CASE WHEN g.enabled THEN 'active' ELSE 'disabled' END AS status,
    COALESCE(fb.page_id, vb.bot_uri, vb.bot_id, 'N/A') AS contact,
    ma.id::text AS provider_app_id,
    g.created_at,
    g.updated_at
FROM im_provider.gates g
LEFT JOIN im_provider.facebook fb ON g.id = fb.gate_id
LEFT JOIN im_provider.meta_apps ma ON fb.meta_app_id = ma.id
LEFT JOIN im_provider.gate_viber vb ON g.id = vb.gate_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Revert to the Facebook-only summary view.
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
