-- +goose Up
-- +goose StatementBegin

ALTER TABLE l2_block ADD COLUMN last_applied_l1_block BIGINT NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS l2_block
DROP COLUMN last_applied_l1_block;

-- +goose StatementEnd
