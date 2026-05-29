-- +goose Up
-- +goose StatementBegin
create table if not exists "im_provider"."bots"(
  "id" uuid primary key references "im_provider"."gates" on delete cascade,
  "sub" text not null check(trim("sub")<>''),
  "iss" text not null check(trim("iss")<>''),
  "updated_at" timestamp with time zone not null default now(),

  unique("iss", "sub")
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists "im_provider"."bots";

-- +goose StatementEnd
