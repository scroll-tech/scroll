-- +goose Up
-- +goose StatementBegin
CREATE TABLE batch_event_v2
(
    id                  BIGSERIAL     PRIMARY KEY,
    l1_block_number     BIGINT        NOT NULL,
    batch_status        SMALLINT      NOT NULL,
    batch_index         BIGINT        NOT NULL,
    batch_hash          VARCHAR       NOT NULL,
    start_block_number  BIGINT        NOT NULL,
    end_block_number    BIGINT        NOT NULL,
    update_status       SMALLINT      NOT NULL,
    created_at          TIMESTAMP(0)  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0)  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0)  DEFAULT NULL
);

CREATE INDEX IF NOT EXISTS idx_be_l1_block_number ON batch_event_v2 (l1_block_number);
CREATE INDEX IF NOT EXISTS idx_be_batch_index ON batch_event_v2 (batch_index);
CREATE INDEX IF NOT EXISTS idx_be_batch_index_batch_hash ON batch_event_v2 (batch_index, batch_hash);
CREATE INDEX IF NOT EXISTS idx_be_end_block_number_update_status_batch_status_batch_index ON batch_event_v2 (end_block_number, update_status, batch_status, batch_index);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS batch_event_v2;
-- +goose StatementEnd