-- +goose Up
-- +goose StatementBegin

ALTER TABLE chunk
ADD COLUMN crc_max VARCHAR NOT NULL,
ADD COLUMN blob_len INTEGER NOT NULL DEFAULT 0;

ALTER TABLE batch
ADD COLUMN data_hash VARCHAR NOT NULL,
ADD COLUMN blob_data_proof BYTEA DEFAULT NULL,
ADD COLUMN blob_len INTEGER NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS batch
DROP COLUMN data_hash,
DROP COLUMN blob_data_proof,
DROP COLUMN blob_len;

ALTER TABLE IF EXISTS chunk
DROP COLUMN crc_max,
DROP COLUMN blob_len;

-- +goose StatementEnd
