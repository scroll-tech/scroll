-- +goose Up
-- +goose StatementBegin

create table l1_block
(
-- block
    number                  BIGINT          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    base_fee                BIGINT          NOT NULL,

-- import
    block_status            SMALLINT        NOT NULL DEFAULT 1,
    import_tx_hash          VARCHAR         DEFAULT NULL,

-- oracle
    oracle_status           SMALLINT        NOT NULL DEFAULT 1,
    oracle_tx_hash          VARCHAR         DEFAULT NULL

-- metadata
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL
);

comment
on column l1_block.block_status is 'undefined, pending, importing, imported, failed';

comment
on column l1_block.oracle_status is 'undefined, pending, importing, imported, failed';

create unique index l1_block_hash_uindex
on l1_block (hash);

create unique index l1_block_number_uindex
on l1_block (number);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l1_block;
-- +goose StatementEnd
