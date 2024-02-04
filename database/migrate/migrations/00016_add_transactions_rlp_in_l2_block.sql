-- +goose Up
-- +goose StatementBegin

ALTER TABLE l2_block
ADD COLUMN transactions_rlp BYTEA NOT NULL DEFAULT '';

ALTER TABLE l2_block
ALTER COLUMN transactions SET DEFAULT '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE l2_block
DROP COLUMN transactions_rlp;

ALTER TABLE l2_block
ALTER COLUMN transactions DROP DEFAULT;

-- +goose StatementEnd
