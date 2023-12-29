-- +goose Up
-- +goose StatementBegin
CREATE TABLE cross_message_v2
(
    id                  BIGSERIAL    PRIMARY KEY,
    message_type        SMALLINT     NOT NULL,
    tx_status           SMALLINT     NOT NULL,
    rollup_status       SMALLINT     NOT NULL,
    token_type          SMALLINT     NOT NULL,
    sender              VARCHAR      NOT NULL,
    receiver            VARCHAR      NOT NULL,

    message_hash        VARCHAR      DEFAULT NULL, -- NULL for failed txs
    l1_tx_hash          VARCHAR      DEFAULT NULL,
    l1_replay_tx_hash   VARCHAR      DEFAULT NULL,
    l1_refund_tx_hash   VARCHAR      DEFAULT NULL,
    l2_tx_hash          VARCHAR      DEFAULT NULL,
    l1_block_number     BIGINT       DEFAULT NULL,
    l2_block_number     BIGINT       DEFAULT NULL,
    l1_token_address    VARCHAR      DEFAULT NULL,
    l2_token_address    VARCHAR      DEFAULT NULL,
    token_ids           VARCHAR      DEFAULT NULL,
    token_amounts       VARCHAR      NOT NULL,
    block_timestamp     BIGINT       NOT NULL,     -- timestamp to sort L1 Deposit & L2 Withdraw events altogether

--- claim info
    message_from        VARCHAR      DEFAULT NULL,
    message_to          VARCHAR      DEFAULT NULL,
    message_value       VARCHAR      DEFAULT NULL,
    message_nonce       BIGINT       DEFAULT NULL,
    message_data        VARCHAR      DEFAULT NULL,
    merkle_proof        BYTEA        DEFAULT NULL,
    batch_index         BIGINT       DEFAULT NULL,

-- metadata
    created_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0) DEFAULT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cm_message_hash ON cross_message_v2 (message_hash);
CREATE INDEX IF NOT EXISTS idx_cm_message_type_l1_block_number ON cross_message_v2 (message_type, l1_block_number DESC);
CREATE INDEX IF NOT EXISTS idx_cm_message_type_l2_block_number ON cross_message_v2 (message_type, l2_block_number DESC);
CREATE INDEX IF NOT EXISTS idx_cm_message_type_rollup_status_message_nonce ON cross_message_v2 (message_type, rollup_status, message_nonce DESC);
CREATE INDEX IF NOT EXISTS idx_cm_message_type_message_nonce_tx_status_l2_block_number ON cross_message_v2 (message_type, message_nonce, tx_status, l2_block_number);
CREATE INDEX IF NOT EXISTS idx_cm_l1_tx_hash ON cross_message_v2 (l1_tx_hash);
CREATE INDEX IF NOT EXISTS idx_cm_l2_tx_hash ON cross_message_v2 (l2_tx_hash);
CREATE INDEX IF NOT EXISTS idx_cm_message_type_tx_status_sender_block_timestamp ON cross_message_v2 (message_type, tx_status, sender, block_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_cm_message_type_sender_block_timestamp ON cross_message_v2 (message_type, sender, block_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_cm_sender_block_timestamp ON cross_message_v2 (sender, block_timestamp DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cross_message_v2;
-- +goose StatementEnd
