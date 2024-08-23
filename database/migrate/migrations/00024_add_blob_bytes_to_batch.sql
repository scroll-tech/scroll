-- +goose Up
-- +goose StatementBegin

ALTER TABLE batch
ADD COLUMN blob_bytes BYTEA;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS batch
DROP COLUMN IF EXISTS blob_bytes;

-- +goose StatementEnd
