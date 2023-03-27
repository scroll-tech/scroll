-- +goose Up
-- +goose StatementBegin

create table block_batch
(
    hash                    VARCHAR         NOT NULL,
    index                   BIGINT          NOT NULL,
    start_block_number      BIGINT          NOT NULL,
    start_block_hash        VARCHAR         NOT NULL,
    end_block_number        BIGINT          NOT NULL,
    end_block_hash          VARCHAR         NOT NULL,
    parent_hash             VARCHAR         NOT NULL,
    state_root              VARCHAR         NOT NULL,
    total_tx_num            BIGINT          NOT NULL,
    total_l1_tx_num         BIGINT          NOT NULL,
    total_l2_gas            BIGINT          NOT NULL,
    proving_status          INTEGER         DEFAULT 1,
    proof                   BYTEA           DEFAULT NULL,
    instance_commitments    BYTEA           DEFAULT NULL,
    proof_time_sec          INTEGER         DEFAULT 0,
    rollup_status           INTEGER         DEFAULT 1,
    commit_tx_hash          VARCHAR         DEFAULT NULL,
    finalize_tx_hash        VARCHAR         DEFAULT NULL,
    oracle_status           INTEGER         DEFAULT 1,
    oracle_tx_hash          VARCHAR         DEFAULT NULL,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    committed_at            TIMESTAMP(0)    DEFAULT NULL,
    finalized_at            TIMESTAMP(0)    DEFAULT NULL
);

comment
on column block_batch.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';
comment
on column block_batch.rollup_status is 'undefined, pending, committing, committed, finalizing, finalized, finalization_skipped, commit_failed, finalize_failed';
comment
on column block_batch.oracle_status is 'undefined, pending, importing, imported, failed';

create unique index block_batch_hash_uindex
    on block_batch (hash);
create unique index block_batch_index_uindex
    on block_batch (index);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists  block_batch;
-- +goose StatementEnd
