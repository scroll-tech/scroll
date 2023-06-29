-- +goose Up
-- +goose StatementBegin

create table l2_block
(
    number                  BIGINT          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    parent_hash             VARCHAR         NOT NULL,
    header                  TEXT            NOT NULL,
    transactions            TEXT            NOT NULL,
    withdraw_trie_root      VARCHAR         NOT NULL,
    tx_num                  INTEGER         NOT NULL,
    gas_used                BIGINT          NOT NULL,
    block_timestamp         NUMERIC         NOT NULL,
    chunk_hash              VARCHAR         DEFAULT NULL
);

create unique index l2_block_hash_uindex
    on l2_block (hash);

create unique index l2_block_number_uindex
    on l2_block (number);

create index l2_block_chunk_hash_index
    on l2_block (chunk_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l2_block;
-- +goose StatementEnd
