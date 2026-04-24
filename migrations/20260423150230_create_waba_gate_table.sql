-- +goose Up
create table if not exists "im_provider"."gate_waba"(
  "id" uuid primary key references "im_provider"."gates" on delete cascade,
  "meta_app_id" uuid not null references "im_provider"."meta_apps" on delete restrict,
  "phone_number" text not null check (trim("phone_number") <> ''),
  "phone_number_id" text not null check (trim("phone_number_id") <> ''),
  "access_token" bytea not null,
  "access_token_expires_at" timestamp with time zone not null,
  "business_id" text not null check (trim("business_id") <> ''),
);

-- +goose Down
drop table if exists "im_provider"."gate_waba";
