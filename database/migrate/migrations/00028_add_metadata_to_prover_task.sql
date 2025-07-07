-- +goose Up
-- +goose StatementBegin

ALTER TABLE prover_task
ADD COLUMN metadata BYTEA;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS prover_task
DROP COLUMN IF EXISTS metadata;

-- +goose StatementEnd