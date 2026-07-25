-- +goose Up
create table if not exists "im_provider"."gate_viber"(
  "gate_id"       uuid primary key references "im_provider"."gates" on delete cascade,
  "bot_id"        text not null check (trim("bot_id") <> ''),   -- Viber account id from get_account_info
  "bot_uri"       text,                                         -- Viber public-account uri
  "auth_token"    text not null,                                -- AES-GCM-encrypted (base64) X-Viber-Auth-Token
  "sender_name"   text not null check (trim("sender_name") <> ''),
  "sender_avatar" text,
  "webhook_uri"   text not null check (trim("webhook_uri") <> ''), -- secret path segment used to resolve the gate on inbound

  unique("bot_id"),
  unique("webhook_uri")
);


-- +goose Down
drop table if exists "im_provider"."gate_viber";
