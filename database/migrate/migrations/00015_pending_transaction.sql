-- +goose Up
-- +goose StatementBegin

CREATE TABLE pending_transaction
(
    id                  SERIAL       PRIMARY KEY,

-- context info
    context_id          VARCHAR      NOT NULL, -- batch hash in commit/finalize tx, block hash in update gas oracle tx
    hash                VARCHAR      NOT NULL,
    status              SMALLINT     NOT NULL,
    rlp_encoding        BYTEA        NOT NULL,

-- debug info
    chain_id            BIGINT       NOT NULL,
    type                SMALLINT     NOT NULL,
    gas_tip_cap         BIGINT       NOT NULL,
    gas_fee_cap         BIGINT       NOT NULL, -- based on geth's implementation, it's gas price in legacy tx.
    gas_limit           BIGINT       NOT NULL,
    nonce               BIGINT       NOT NULL,
    submit_block_number BIGINT       NOT NULL,

-- sender info
    sender_name         VARCHAR      NOT NULL,
    sender_service      VARCHAR      NOT NULL,
    sender_address      VARCHAR      NOT NULL,
    sender_type         SMALLINT     NOT NULL,

    created_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0) DEFAULT NULL
);

CREATE UNIQUE INDEX unique_idx_pending_transaction_on_hash ON pending_transaction(hash);
CREATE INDEX idx_pending_transaction_on_sender_type_status_nonce_gas_fee_cap ON pending_transaction (sender_type, status, nonce, gas_fee_cap);
CREATE INDEX idx_pending_transaction_on_sender_address_nonce ON pending_transaction(sender_address, nonce);

COMMENT ON COLUMN pending_transaction.sender_type IS 'unknown, commit batch, finalize batch, L1 gas oracle, L2 gas oracle';
COMMENT ON COLUMN pending_transaction.status IS 'unknown, pending, replaced, confirmed, confirmed failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS pending_transaction;
-- +goose StatementEnd
