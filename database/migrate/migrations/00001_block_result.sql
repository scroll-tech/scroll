-- +goose Up
-- +goose StatementBegin

-- TODO: use foreign key for batch_id?
-- TODO: why tx_num is bigint?
create table block_result
(
    number                  BIGINT          not null,
    hash                    VARCHAR         not null,
    parent_hash             VARCHAR         default null,
    content                 json            not null,
    batch_id                VARCHAR         default null,
    tx_num                  INTEGER         NOT NULL DEFAULT 0,
    gas_used                BIGINT          NOT NULL DEFAULT 0,
    block_timestamp         NUMERIC         NOT NULL DEFAULT 0
);

create unique index block_result_hash_uindex
    on block_result (hash);

create unique index block_result_number_uindex
    on block_result (number);

create unique index block_result_parent_uindex
    on block_result (number, parent_hash);

create unique index block_result_parent_hash_uindex
    on block_result (hash, parent_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists block_result;
-- +goose StatementEnd
