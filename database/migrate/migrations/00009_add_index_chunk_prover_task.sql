-- +goose Up
-- +goose StatementBegin

create index if not exists idx_chunk_hash on chunk(hash, deleted_at) where deleted_at IS NULL;

create index if not exists idx_proving_status_end_block_number_index on chunk(index, end_block_number, proving_status, deleted_at) where deleted_at IS NULL;

create index if not exists  idx_publickey_proving_status on prover_task(prover_public_key, proving_status, deleted_at, id) where deleted_at is null;

-- +goose StatementEnd
