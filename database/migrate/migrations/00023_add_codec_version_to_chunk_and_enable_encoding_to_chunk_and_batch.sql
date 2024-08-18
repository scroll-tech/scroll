-- +goose Up
-- +goose StatementBegin

ALTER TABLE chunk
ADD COLUMN codec_version SMALLINT NOT NULL DEFAULT -1,
ADD COLUMN enable_encode BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE batch
ADD COLUMN enable_encode BOOLEAN NOT NULL DEFAULT false;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS chunk
DROP COLUMN IF EXISTS enable_encode,
DROP COLUMN IF EXISTS codec_version;

ALTER TABLE IF EXISTS batch
DROP COLUMN IF EXISTS enable_encode;

-- +goose StatementEnd
