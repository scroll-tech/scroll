-- +goose Up
-- +goose StatementBegin

ALTER TABLE batch
ADD COLUMN challenge_digest VARCHAR DEFAULT '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS batch
DROP COLUMN IF EXISTS challenge_digest;

-- +goose StatementEnd
