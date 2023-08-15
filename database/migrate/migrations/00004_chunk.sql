-- +goose Up
-- +goose StatementBegin

create table chunk
(
-- chunk
    index                             BIGINT          NOT NULL,
    hash                              VARCHAR         NOT NULL,
    start_block_number                BIGINT          NOT NULL,
    start_block_hash                  VARCHAR         NOT NULL,
    end_block_number                  BIGINT          NOT NULL,
    end_block_hash                    VARCHAR         NOT NULL,
    total_l1_messages_popped_before   BIGINT          NOT NULL,
    total_l1_messages_popped_in_chunk INTEGER         NOT NULL,
    start_block_time                  BIGINT          NOT NULL,
    parent_chunk_hash                 VARCHAR         NOT NULL,
    state_root                        VARCHAR         NOT NULL,
    parent_chunk_state_root           VARCHAR         NOT NULL,
    withdraw_root                     VARCHAR         NOT NULL,

-- proof
    proving_status                    SMALLINT        NOT NULL DEFAULT 1,
    proof                             BYTEA           DEFAULT NULL,
    prover_assigned_at                TIMESTAMP(0)    DEFAULT NULL, -- DEPRECATED
    proved_at                         TIMESTAMP(0)    DEFAULT NULL,
    proof_time_sec                    INTEGER         DEFAULT NULL,

-- batch
    batch_hash                        VARCHAR         DEFAULT NULL,

-- metadata
    total_l2_tx_gas                   BIGINT          NOT NULL,
    total_l2_tx_num                   INTEGER         NOT NULL,
    total_l1_commit_calldata_size     INTEGER         NOT NULL,
    total_l1_commit_gas               BIGINT          NOT NULL,
    created_at                        TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                        TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at                        TIMESTAMP(0)    DEFAULT NULL
);

comment
on column chunk.proving_status is 'undefined, unassigned, assigned, proved, verified, failed';

create unique index chunk_index_uindex
on chunk (index) where deleted_at IS NULL;

create unique index chunk_hash_uindex
on chunk (hash) where deleted_at IS NULL;

create index batch_hash_index
on chunk (batch_hash) where deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists chunk;
-- +goose StatementEnd
