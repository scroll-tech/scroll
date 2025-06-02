-- +goose Up
-- +goose StatementBegin

CREATE TABLE blob_upload (
    batch_index     BIGINT          NOT NULL,

    platform        SMALLINT        NOT NULL,
    status          SMALLINT        NOT NULL,

-- metadata
    updated_at      TIMESTAMP       NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMP(0)    DEFAULT NULL,

    PRIMARY KEY (batch_index, platform),
    FOREIGN KEY (batch_index) REFERENCES batch(index)
);

COMMENT ON COLUMN blob_upload.status IS 'undefined, pending, uploaded, failed';

CREATE INDEX IF NOT EXISTS idx_blob_upload_batch_index ON blob_upload(batch_index) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_blob_upload_platform ON blob_upload(platform) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_blob_upload_status ON blob_upload(status) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_blob_upload_updated_at ON blob_upload(updated_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_blob_upload_status_platform ON blob_upload(status, platform) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_blob_upload_batch_index_status_platform ON blob_upload(batch_index, status, platform) WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE blob_upload;
-- +goose StatementEnd