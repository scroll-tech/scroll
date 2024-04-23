-- +goose Up
-- +goose StatementBegin

ALTER TABLE l1_block
ADD COLUMN blob_base_fee BIGINT DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS l1_block
DROP COLUMN blob_base_fee;

-- +goose StatementEnd
