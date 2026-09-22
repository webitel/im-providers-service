-- +goose Up
create table if not exists "im_provider"."gate_custom"(
  "gate_id"            uuid primary key references "im_provider"."gates" on delete cascade,
  "callback_url"       text not null check (trim("callback_url") <> ''),      -- customer-side webhook operator replies are delivered to
  "app_secret"         text not null check (trim("app_secret") <> ''),        -- AES-GCM-encrypted (base64) X-Webitel-Sign shared secret
  "webhook_uri"        text not null check (trim("webhook_uri") <> ''),       -- secret path segment used to resolve the gate on inbound
  "allowed_ips"        text[] not null default '{}',                          -- CIDR allowlist; empty accepts any source address
  "request_timeout_ms" integer  not null default 5000 check ("request_timeout_ms" between 500 and 60000),
  "retry_attempts"     smallint not null default 3 check ("retry_attempts" between 0 and 10),

  unique("webhook_uri")
);


-- +goose Down
drop table if exists "im_provider"."gate_custom";
