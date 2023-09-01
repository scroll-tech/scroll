-- +goose Up
-- +goose StatementBegin

ALTER TABLE batch
ADD COLUMN total_l1_commit_gas BIGINT NOT NULL DEFAULT 0,
ADD COLUMN total_l1_commit_calldata_size INTEGER NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS batch
DROP COLUMN total_l1_commit_gas,
DROP COLUMN total_l1_commit_calldata_size;

-- +goose StatementEnd
