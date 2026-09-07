-- +goose Up
-- +goose StatementBegin

-- Maps the conversation id owned by the external system to the external user it
-- belongs to. Outbound needs it to address a reply; a recipient with no row yet
-- has never written to us, so the reply goes out as an operator-initiated
-- broadcast instead of a message.
create table if not exists "im_provider"."gate_custom_chats"(
  "gate_id"      uuid not null references "im_provider"."gates" on delete cascade,
  "chat_id"      text not null check (trim("chat_id") <> ''),
  "external_sub" text not null check (trim("external_sub") <> ''), -- "{sender.type}|{sender.id}"
  "closed_at"    timestamptz,
  "created_at"   timestamptz not null default now(),
  "updated_at"   timestamptz not null default now(),

  primary key ("gate_id", "chat_id")
);

create unique index if not exists "gate_custom_chats_gate_sub_idx"
  on "im_provider"."gate_custom_chats" ("gate_id", "external_sub");

create trigger tr_gate_custom_chats_updated
before update on "im_provider"."gate_custom_chats"
for each row execute function im_provider.update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop table if exists "im_provider"."gate_custom_chats";

-- +goose StatementEnd
