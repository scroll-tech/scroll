-- +goose Up
-- +goose StatementBegin

-- TODO: use foreign key for batch_id?
-- TODO: why tx_num is bigint?
create table block_tx
(
    number                  BIGINT          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    parent_hash             VARCHAR         NOT NULL,
    tx                      JSON            NOT NULL,
    chunk_hash              VARCHAR         DEFAULT NULL,
    tx_num                  INTEGER         NOT NULL,
    gas_used                BIGINT          NOT NULL,
    block_timestamp         NUMERIC         NOT NULL
);

create unique index block_tx_hash_uindex
    on block_tx (hash);

create unique index block_tx_number_uindex
    on block_tx (number);

create unique index block_tx_parent_uindex
    on block_tx (number, parent_hash);

create unique index block_tx_parent_hash_uindex
    on block_tx (hash, parent_hash);

create index block_tx_batch_hash_index
    on block_tx (batch_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists block_tx;
-- +goose StatementEnd
