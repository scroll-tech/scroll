-- +goose Up
-- +goose StatementBegin

-- Add index on status for faster filtering by status
CREATE INDEX idx_blob_upload_status ON blob_upload(status);

-- Add index on updated_at for faster sorting and filtering by time
CREATE INDEX idx_blob_upload_updated_at ON blob_upload(updated_at);

-- Add index on (batch_index, status) for faster filtering by both fields
CREATE INDEX idx_blob_upload_batch_index_status ON blob_upload(batch_index, status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_blob_upload_status;
DROP INDEX IF EXISTS idx_blob_upload_updated_at;
DROP INDEX IF EXISTS idx_blob_upload_batch_index_status;
-- +goose StatementEnd 