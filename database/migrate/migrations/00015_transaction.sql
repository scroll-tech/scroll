-- +goose Up
-- +goose StatementBegin

CREATE TABLE transaction
(
    id                  SERIAL       PRIMARY KEY,

    context_id          VARCHAR      NOT NULL, -- batch hash in commit/finalize tx, block hash in update gas oracle tx
    hash                VARCHAR      NOT NULL,
    type                SMALLINT     NOT NULL,
    status              SMALLINT     NOT NULL,
    rlp_encoding        BYTEA        NOT NULL,

    gas_fee_cap         NUMERIC      NOT NULL,
    gas_tip_cap         NUMERIC      NOT NULL,
    gas_price           NUMERIC      NOT NULL,
    gas_limit           BIGINT       NOT NULL,
    nonce               BIGINT       NOT NULL,
    submit_at           BIGINT       NOT NULL,           

    sender_name         VARCHAR      NOT NULL,
    sender_service      VARCHAR      NOT NULL,
    sender_address      VARCHAR      NOT NULL,
    sender_type         SMALLINT     NOT NULL,

    created_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0) DEFAULT NULL
);

CREATE INDEX idx_transaction_on_context_id
ON transaction (context_id);

CREATE INDEX idx_transaction_on_sender_type_status_nonce
ON transaction (sender_type, status, nonce);

COMMENT ON COLUMN transaction.type IS 'unknown, commit batch, finalize batch, L1 gas oracle, L2 gas oracle';

COMMENT ON COLUMN transaction.status IS 'unknown, pending, confirmed, failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction;
-- +goose StatementEnd