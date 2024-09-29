-- +goose Up
-- +goose StatementBegin

ALTER TABLE chunk
ADD COLUMN codec_version SMALLINT NOT NULL DEFAULT 0,
ADD COLUMN enable_compress BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE batch
ADD COLUMN enable_compress BOOLEAN NOT NULL DEFAULT false;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS chunk
DROP COLUMN IF EXISTS enable_compress,
DROP COLUMN IF EXISTS codec_version;

ALTER TABLE IF EXISTS batch
DROP COLUMN IF EXISTS enable_compress;

-- +goose StatementEnd
