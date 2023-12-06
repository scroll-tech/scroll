-- +goose Up
-- +goose StatementBegin
create table cross_message
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

CREATE UNIQUE INDEX if not exists idx_cm_message_hash ON cross_message (message_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists cross_message;
-- +goose StatementEnd
