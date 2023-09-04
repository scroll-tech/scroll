-- +goose Up
-- +goose StatementBegin
drop index if exists l1_message_hash_uindex;

create index if not exists l1_message_hash_index
on l1_message (msg_hash) where deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists l1_message_hash_index;

create unique index if not exists l1_message_hash_uindex
on l1_message (msg_hash) where deleted_at IS NULL;
-- +goose StatementEnd
