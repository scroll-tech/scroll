-- +goose Up
-- +goose StatementBegin

ALTER TABLE batch
ADD COLUMN data_hash VARCHAR NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS batch
DROP COLUMN data_hash;

-- +goose StatementEnd
