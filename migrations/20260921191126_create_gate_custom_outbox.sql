-- +goose Up
-- +goose StatementBegin

create table if not exists "im_provider"."gate_custom_outbox"(
  "id"              bigserial primary key,
  "gate_id"         uuid not null references "im_provider"."gates" on delete cascade,
  "chat_key"        text not null check (trim("chat_key") <> ''),
  "message_id"      text not null check (trim("message_id") <> ''),
  "payload"         bytea not null,
  "attempt"         smallint not null default 0,
  "status"          text not null default 'pending' check ("status" in ('pending', 'failed')),
  "next_attempt_at" timestamptz not null default now(),
  "locked_until"    timestamptz not null default '-infinity',
  "last_error"      text,
  "created_at"      timestamptz not null default now(),
  "updated_at"      timestamptz not null default now()
);

create index if not exists "gate_custom_outbox_due_idx"
  on "im_provider"."gate_custom_outbox" ("next_attempt_at")
  where "status" = 'pending';

create index if not exists "gate_custom_outbox_chat_idx"
  on "im_provider"."gate_custom_outbox" ("gate_id", "chat_key")
  where "status" = 'pending';

create trigger tr_gate_custom_outbox_updated
before update on "im_provider"."gate_custom_outbox"
for each row execute function im_provider.update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop table if exists "im_provider"."gate_custom_outbox";

-- +goose StatementEnd
