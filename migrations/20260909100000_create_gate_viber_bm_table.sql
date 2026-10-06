-- +goose Up
create table if not exists "im_provider"."gate_viber_bm"(
  "gate_id"        uuid primary key references "im_provider"."gates" on delete cascade,
  "base_url"       text not null check (trim("base_url") <> ''),        -- per-account InfoBip host
  "sender_name"    text not null check (trim("sender_name") <> ''),     -- registered Viber sender == inbound `to`
  "api_key"        text not null,                                       -- AES-GCM encrypted (base64) InfoBip App key
  "webhook_secret" text,                                                -- AES-GCM encrypted inbound Authorization credential (nullable = no check)
  "webhook_uri"    text not null check (trim("webhook_uri") <> ''),     -- secret path segment resolving the gate

  unique("sender_name"),
  unique("webhook_uri")
);


-- +goose Down
drop table if exists "im_provider"."gate_viber_bm";
