-- +goose Up
-- +goose StatementBegin

ALTER TABLE chunk
    ADD COLUMN total_attempts SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN active_attempts SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE batch
    ADD COLUMN total_attempts SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN active_attempts SMALLINT NOT NULL DEFAULT 0;

create index if not exists idx_total_attempts_active_attempts_end_block_number
    on chunk (total_attempts, active_attempts, end_block_number)
    where deleted_at IS NULL;

create index if not exists idx_total_attempts_active_attempts_chunk_proofs_status
    on batch (total_attempts, active_attempts, chunk_proofs_status)
    where deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop index if exists idx_total_attempts_active_attempts_end_block_number;
drop index if exists idx_total_attempts_active_attempts_chunk_proofs_status;

-- +goose StatementEnd