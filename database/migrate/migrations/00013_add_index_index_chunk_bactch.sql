-- +goose Up
-- +goose StatementBegin

create index if not exists idx_chunk_index on chunk(index) where deleted_at IS NULL;

create index if not exists idx_batch_index on batch(index) where deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop index if exists idx_chunk_index;
drop index if exists idx_batch_index;

-- +goose StatementEnd
