-- +goose Up
-- +goose StatementBegin

drop index if exists idx_total_attempts_active_attempts_end_block_number;
drop index if exists idx_total_attempts_active_attempts_chunk_proofs_status;

create index if not exists idx_chunk_proving_status_index on chunk (proving_status, index) where deleted_at IS NULL;
create index if not exists idx_batch_proving_status_index on batch (proving_status, chunk_proofs_status, index) where deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

create index if not exists idx_total_attempts_active_attempts_end_block_number
    on chunk (total_attempts, active_attempts, end_block_number)
    where deleted_at IS NULL;

create index if not exists idx_total_attempts_active_attempts_chunk_proofs_status
    on batch (total_attempts, active_attempts, chunk_proofs_status)
    where deleted_at IS NULL;


drop index if exists idx_chunk_proving_status_index;
drop index if exists idx_batch_proving_status_index;

-- +goose StatementEnd
