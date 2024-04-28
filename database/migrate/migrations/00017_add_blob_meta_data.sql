-- +goose Up
-- +goose StatementBegin

ALTER TABLE chunk
ADD COLUMN crc_max INTEGER DEFAULT 0,
ADD COLUMN blob_size INTEGER DEFAULT 0;

ALTER TABLE batch
ADD COLUMN data_hash VARCHAR DEFAULT '',
ADD COLUMN blob_data_proof BYTEA DEFAULT NULL,
ADD COLUMN blob_size INTEGER DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS batch
DROP COLUMN data_hash,
DROP COLUMN blob_data_proof,
DROP COLUMN blob_size;

ALTER TABLE IF EXISTS chunk
DROP COLUMN crc_max,
DROP COLUMN blob_size;

-- +goose StatementEnd
