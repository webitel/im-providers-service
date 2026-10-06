-- +goose Up
-- Inbound routing resolves a gate by (sender_name, webhook_uri) and webhook_uri
-- is already unique; several accounts can share one sender (Infobip trial
-- "IBSelfServe"), so sender_name alone must not be unique.
alter table "im_provider"."gate_viber_bm" drop constraint if exists "gate_viber_bm_sender_name_key";

-- +goose Down
alter table "im_provider"."gate_viber_bm" add constraint "gate_viber_bm_sender_name_key" unique ("sender_name");
