-- +goose Up
create table if not exists "im_provider"."binded_contact"(
  "id" uuid primary key references "im_provider"."gates"("id") on delete cascade,
  "sub" text not null check (trim("sub")<>''),
  "iss" text not null check(trim("iss")<>'')
);

-- +goose Down
drop table if exists "im_provider"."binded_contact";
