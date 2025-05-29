-- +goose Up
-- +goose StatementBegin

CREATE TABLE blob_upload (
    batch_index     BIGINT     NOT NULL,

    platform        TEXT       NOT NULL,
    status          SMALLINT   NOT NULL,
    updated_at      TIMESTAMP  NOT NULL DEFAULT now(),

    PRIMARY KEY (batch_index, platform),
    FOREIGN KEY (batch_index) REFERENCES batch(index)
);

COMMENT ON COLUMN blob_upload.status IS 'undefined, pending, uploaded, failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE blob_upload;
-- +goose StatementEnd