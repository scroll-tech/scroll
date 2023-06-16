-- +goose Up
-- +goose StatementBegin

create table chunk
(
-- block
    index                           BIGINT          NOT NULL,
    hash                            VARCHAR         NOT NULL,
    start_block_number              BIGINT          NOT NULL,
    start_block_hash                VARCHAR         NOT NULL,
    end_block_number                BIGINT          NOT NULL,
    end_block_hash                  VARCHAR         NOT NULL,
    total_gas_used                  BIGINT          NOT NULL,
	total_tx_num                    BIGINT          NOT NULL,
    total_payload_size              BIGINT          NOT NULL,
    total_l1_messages_popped_before BIGINT          NOT NULL,
    total_l1_messages               BIGINT          NOT NULL,

-- proof
    proving_status                  SMALLINT        NOT NULL DEFAULT 1,
    proof                           BYTEA           DEFAULT NULL,
    proof_time_sec                  SMALLINT        DEFAULT NULL,
    prover_assigned_at              TIMESTAMP(0)    DEFAULT NULL,
    proved_at                       TIMESTAMP(0)    DEFAULT NULL,

-- batch
    batch_hash                      VARCHAR         DEFAULT NULL,

-- metadata
    created_at                      TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                      TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at                      TIMESTAMP(0)    DEFAULT NULL
);

comment
on column chunk.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';

create unique index chunk_index_uindex
on chunk (index);

create unique index chunk_hash_uindex
on chunk (hash);

create index batch_hash_index
on chunk (batch_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists chunk;
-- +goose StatementEnd
