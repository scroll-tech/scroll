-- +goose Up
-- +goose StatementBegin

create table transaction
(
    hash         VARCHAR      NOT NULL,
    tx_hash      VARCHAR      NOT NULL,
    sender       VARCHAR      NOT NULL,
    nonce        BIGINT       NOT NULL,
    target       VARCHAR      DEFAULT '',
    value        VARCHAR      NOT NULL,
    data         BYTEA        DEFAULT NULL,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create unique index transaction_hash_uindex
    on transaction (hash);

create unique index transaction_tx_hash_uindex
    on transaction (tx_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists transaction;
-- +goose StatementEnd