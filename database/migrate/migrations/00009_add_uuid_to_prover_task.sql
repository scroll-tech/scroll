-- +goose Up
-- +goose StatementBegin
ALTER TABLE prover_task ADD COLUMN uuid uuid DEFAULT gen_random_uuid() NOT NULL UNIQUE;

create index if not exists idx_uuid on prover_task (uuid) where deleted_at IS NULL;
-- +goose StatementEnd
