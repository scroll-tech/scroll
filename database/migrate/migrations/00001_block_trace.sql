-- +goose Up
-- +goose StatementBegin

-- TODO: use foreign key for batch_id?
-- TODO: why tx_num is bigint?
create table block_trace
(
    number                  BIGINT          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    parent_hash             VARCHAR         DEFAULT NULL,
    trace                   JSON            NOT NULL,
    batch_id                VARCHAR         DEFAULT NULL,
    tx_num                  INTEGER         NOT NULL DEFAULT 0,
    gas_used                BIGINT          NOT NULL DEFAULT 0,
    block_timestamp         NUMERIC         NOT NULL DEFAULT 0
);

create unique index block_trace_hash_uindex
    on block_trace (hash);

create unique index block_trace_number_uindex
    on block_trace (number);

create unique index block_trace_parent_uindex
    on block_trace (number, parent_hash);

create unique index block_trace_parent_hash_uindex
    on block_trace (hash, parent_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists block_trace;
-- +goose StatementEnd
