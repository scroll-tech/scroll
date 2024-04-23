-- +goose Up
-- +goose StatementBegin

DROP INDEX if exists idx_prover_block_list_on_public_key;

CREATE UNIQUE INDEX if not exists uniq_prover_block_list_on_public_key ON prover_block_list(public_key) where deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

CREATE INDEX if not exists idx_prover_block_list_on_public_key ON prover_block_list(public_key);

DROP INDEX if exists uniq_prover_block_list_on_public_key;

-- +goose StatementEnd
