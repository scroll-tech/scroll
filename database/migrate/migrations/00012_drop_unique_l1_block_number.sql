-- +goose Up
-- +goose StatementBegin
drop index if exists l1_block_number_uindex;

create index l1_block_number_index
on l1_block (number) where deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists l1_block_number_index;

create unique index if not exists l1_block_number_uindex
on l1_block (number) where deleted_at IS NULL;
-- +goose StatementEnd
