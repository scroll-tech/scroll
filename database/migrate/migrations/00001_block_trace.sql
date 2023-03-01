-- +goose Up
-- +goose StatementBegin

-- TODO: use foreign key for batch_id?
-- TODO: why tx_num is bigint?
create table block_trace
(
    number                  BIGINT          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    parent_hash             VARCHAR         NOT NULL,
    trace                   JSON            NOT NULL,
    batch_hash              VARCHAR         DEFAULT NULL,
    tx_num                  INTEGER         NOT NULL,
    gas_used                BIGINT          NOT NULL,
    block_timestamp         NUMERIC         NOT NULL
);

create unique index block_trace_hash_uindex
    on block_trace (hash);

create unique index block_trace_number_uindex
    on block_trace (number);

create unique index block_trace_parent_uindex
    on block_trace (number, parent_hash);

create unique index block_trace_parent_hash_uindex
    on block_trace (hash, parent_hash);

create index block_trace_batch_hash_index
    on block_trace (batch_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists block_trace;
-- +goose StatementEnd
