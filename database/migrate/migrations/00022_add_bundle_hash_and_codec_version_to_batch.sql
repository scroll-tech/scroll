-- +goose Up
-- +goose StatementBegin

ALTER TABLE batch
ADD COLUMN bundle_hash VARCHAR DEFAULT '',
ADD COLUMN codec_version SMALLINT DEFAULT 0;

CREATE INDEX idx_batch_bundle_hash ON batch(bundle_hash);
CREATE INDEX idx_batch_index_codec_version ON batch(index, codec_version);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_batch_bundle_hash;
DROP INDEX IF EXISTS idx_batch_index_codec_version;

ALTER TABLE IF EXISTS batch
DROP COLUMN IF EXISTS bundle_hash,
DROP COLUMN IF EXISTS codec_version;

-- +goose StatementEnd
