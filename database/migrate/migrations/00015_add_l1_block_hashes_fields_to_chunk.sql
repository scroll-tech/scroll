-- +goose Up
-- +goose StatementBegin

ALTER TABLE chunk
    ADD COLUMN last_applied_l1_block BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN l1_block_range_hash VARCHAR DEFAULT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS chunk
DROP COLUMN last_applied_l1_block;
DROP COLUMN l1_block_range_hash;

-- +goose StatementEnd
