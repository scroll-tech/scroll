-- +goose StatementBegin

ALTER TABLE blob_upload
ADD COLUMN tx_hash VARCHAR DEFAULT '',


CREATE INDEX idx_blob_upload_transaction_hash ON blob_upload(tx_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_blob_upload_tx_hash;

ALTER TABLE IF EXISTS blob_upload
DROP COLUMN IF EXISTS tx_hash;

-- +goose StatementEnd
