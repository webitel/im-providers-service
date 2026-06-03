-- +goose Up
-- +goose StatementBegin
alter table "im_provider"."bots" drop constraint if exists "bots_sub_key";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table "im_provider"."bots" add constraint "bots_sub_key" unique ("sub");
-- +goose StatementEnd
