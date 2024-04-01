-- +goose Up
-- +goose StatementBegin

ALTER TABLE batch
ADD COLUMN data_hash VARCHAR NOT NULL,
ADD COLUMN blob_data_proof BYTEA DEFAULT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS batch
DROP COLUMN data_hash,
DROP COLUMN blob_data_proof;

-- +goose StatementEnd
