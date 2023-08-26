-- +goose Up
-- +goose StatementBegin

create index if not exists idx_hash on chunk(hash, deleted_at) where deleted_at IS NULL;

create index if not exists idx_proving_status_end_block_number_index on chunk(proving_status, end_block_number,deleted_at, index) where deleted_at IS NULL;

create index if not exists  idx_publickey_proving_status on prover_task(prover_public_key, proving_status, deleted_at, id) where deleted_at is null;

-- +goose StatementEnd
