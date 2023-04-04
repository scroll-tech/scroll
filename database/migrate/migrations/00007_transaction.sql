-- +goose Up
-- +goose StatementBegin

create table transaction
(
    id           VARCHAR      NOT NULL,
    tx_hash      VARCHAR      NOT NULL,
    sender       VARCHAR      NOT NULL,
    nonce        BIGINT       NOT NULL,
    target       VARCHAR      DEFAULT '',
    value        VARCHAR      NOT NULL,
    data         BYTEA        DEFAULT NULL,
    type         INTEGER      DEFAULT 0,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create unique index transaction_id_uindex
    on transaction (id);

create unique index transaction_tx_hash_uindex
    on transaction (tx_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists transaction;
-- +goose StatementEnd