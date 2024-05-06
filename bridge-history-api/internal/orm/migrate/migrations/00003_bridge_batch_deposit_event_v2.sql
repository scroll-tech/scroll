-- +goose Up
-- +goose StatementBegin
CREATE TABLE bridge_batch_deposit_event_v2
(
    id                  BIGSERIAL    PRIMARY KEY,
    token_type          SMALLINT     NOT NULL,
    sender              VARCHAR      NOT NULL,
    batch_index         BIGINT       DEFAULT NULL,
    token_amount        VARCHAR      NOT NULL,
    fee                 VARCHAR      NOT NULL,
    l1_token_address    VARCHAR      DEFAULT NULL,
    l2_token_address    VARCHAR      DEFAULT NULL,
    l1_block_number     BIGINT       DEFAULT NULL,
    l2_block_number     BIGINT       DEFAULT NULL,
    l1_tx_hash          VARCHAR      DEFAULT NULL,
    l2_tx_hash          VARCHAR      DEFAULT NULL,
    l1_log_index        INTEGER      DEFAULT NULL,
    l2_log_index        INTEGER      DEFAULT NULL,
    tx_status           SMALLINT     NOT NULL,
    block_timestamp     BIGINT       NOT NULL,
    created_at          TIMESTAMP(0)  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0)  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0)  DEFAULT NULL
);

CREATE UNIQUE INDEX idx_l1hash_l1logindex ON bridge_batch_deposit_event_v2 (l1_tx_hash, l1_log_index);
CREATE INDEX IF NOT EXISTS idx_bbde_batchidx_sender ON bridge_batch_deposit_event_v2 (batch_index, sender);
CREATE INDEX IF NOT EXISTS idx_bbde_l1_block_number ON bridge_batch_deposit_event_v2 (l1_block_number DESC);
CREATE INDEX IF NOT EXISTS idx_bbde_l2_block_number ON bridge_batch_deposit_event_v2 (l2_block_number DESC);
CREATE INDEX IF NOT EXISTS idx_bbde_l1_tx_hash ON bridge_batch_deposit_event_v2 (l1_tx_hash DESC);
CREATE INDEX IF NOT EXISTS idx_bbde_sender_block_timestamp ON bridge_batch_deposit_event_v2 (sender, block_timestamp DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bridge_batch_deposit_event_v2;
-- +goose StatementEnd
