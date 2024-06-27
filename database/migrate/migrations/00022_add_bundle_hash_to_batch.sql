-- +goose Up
-- +goose StatementBegin

ALTER TABLE batch
ADD COLUMN bundle_hash VARCHAR DEFAULT '';  -- Adding bundle hash for SQL query consistency and ease of use

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS batch
DROP COLUMN bundle_hash;

-- +goose StatementEnd
