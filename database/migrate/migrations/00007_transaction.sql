-- +goose Up
-- +goose StatementBegin

create table scroll_transaction
(
    id           VARCHAR      NOT NULL,
    tx_hash      VARCHAR      NOT NULL,
    sender       VARCHAR      NOT NULL,
    nonce        BIGINT       NOT NULL,
    target       VARCHAR      DEFAULT '',
    value        VARCHAR      NOT NULL,
    data         BYTEA        DEFAULT NULL,
    type         INTEGER      DEFAULT 0,
    confirm      boolean      DEFAULT false,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create unique index scroll_transaction_id_uindex
    on scroll_transaction (id);

create unique index scroll_transaction_tx_hash_uindex
    on scroll_transaction (tx_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists scroll_transaction;
-- +goose StatementEnd