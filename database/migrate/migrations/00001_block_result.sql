-- +goose Up
-- +goose StatementBegin

create table block_result
(
    number                  BIGINT          not null,
    hash                    varchar         not null,
    content                 json            not null,
    batch_id                integer         default null, -- TODO: foreign key?
    tx_num                  integer         NOT NULL DEFAULT 0, -- FIXME: why tx_num is bigint?
    gas_used                BIGINT          NOT NULL DEFAULT 0,
    block_timestamp         NUMERIC         NOT NULL DEFAULT 0,
);

create unique index block_result_hash_uindex
    on block_result (hash);

create unique index block_result_number_uindex
    on block_result (number);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists  block_result;
-- +goose StatementEnd
