-- +goose Up
-- +goose StatementBegin

CREATE TABLE blob_upload (
    batch_index     BIGINT          NOT NULL,
    batch_hash      VARCHAR         NOT NULL,

    platform        SMALLINT        NOT NULL,
    status          SMALLINT        NOT NULL,

-- metadata
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS batch_index_batch_hash_platform_uindex
ON blob_upload(batch_index, batch_hash, platform) WHERE deleted_at IS NULL;

COMMENT ON COLUMN blob_upload.status IS 'undefined, pending, uploaded, failed';

CREATE INDEX IF NOT EXISTS idx_blob_upload_status_platform ON blob_upload(status, platform) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_blob_upload_batch_index_batch_hash_status_platform 
ON blob_upload(batch_index, batch_hash, status, platform) WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE blob_upload;
-- +goose StatementEnd