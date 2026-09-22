-- +goose Up
-- +goose StatementBegin

DROP VIEW IF EXISTS im_provider.gate_summary;

CREATE VIEW im_provider.gate_summary AS
SELECT
    g.id,
    g.name,
    g.type,
    CASE WHEN g.enabled THEN 'active' ELSE 'disabled' END AS status,
    COALESCE(fb.page_id, vb.bot_uri, vb.bot_id, ig.business_account_id, cu.callback_url, b.sub, 'N/A') AS contact,
    COALESCE(ma.id::text, ma2.id::text) AS provider_app_id,
    g.created_at,
    g.updated_at
FROM im_provider.gates g
LEFT JOIN im_provider.facebook fb ON g.id = fb.gate_id
LEFT JOIN im_provider.meta_apps ma ON fb.meta_app_id = ma.id
LEFT JOIN im_provider.gate_viber vb ON g.id = vb.gate_id
LEFT JOIN im_provider.instagram ig ON g.id = ig.gate_id
LEFT JOIN im_provider.meta_apps ma2 ON ig.meta_app_id = ma2.id
LEFT JOIN im_provider.gate_custom cu ON g.id = cu.gate_id
LEFT JOIN im_provider.bots b ON g.id = b.gate_id AND g.type = 'telegram';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP VIEW IF EXISTS im_provider.gate_summary;

CREATE VIEW im_provider.gate_summary AS
SELECT
    g.id,
    g.name,
    g.type,
    CASE WHEN g.enabled THEN 'active' ELSE 'disabled' END AS status,
    COALESCE(fb.page_id, vb.bot_uri, vb.bot_id, ig.business_account_id, b.sub, 'N/A') AS contact,
    COALESCE(ma.id::text, ma2.id::text) AS provider_app_id,
    g.created_at,
    g.updated_at
FROM im_provider.gates g
LEFT JOIN im_provider.facebook fb ON g.id = fb.gate_id
LEFT JOIN im_provider.meta_apps ma ON fb.meta_app_id = ma.id
LEFT JOIN im_provider.gate_viber vb ON g.id = vb.gate_id
LEFT JOIN im_provider.instagram ig ON g.id = ig.gate_id
LEFT JOIN im_provider.meta_apps ma2 ON ig.meta_app_id = ma2.id
LEFT JOIN im_provider.bots b ON g.id = b.gate_id AND g.type = 'telegram';

-- +goose StatementEnd
