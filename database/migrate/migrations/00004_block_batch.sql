-- +goose Up
-- +goose StatementBegin

create table block_batch
(
    id                      VARCHAR         NOT NULL,
    index                   BIGINT          NOT NULL,
    start_block_number      BIGINT          NOT NULL,
    start_block_hash        VARCHAR         NOT NULL,
    end_block_number        BIGINT          NOT NULL,
    end_block_hash          VARCHAR         NOT NULL,
    parent_hash             VARCHAR         NOT NULL,
    total_tx_num            BIGINT          NOT NULL DEFAULT 0,
    total_l2_gas            BIGINT          NOT NULL DEFAULT 0,
    proving_status          integer         default 1,
    proof                   BYTEA           default null,
    instance_commitments    BYTEA           default null,
    proof_time_sec          integer         default 0,
    rollup_status           integer         default 1,
    commit_tx_hash          varchar         default null,
    finalize_tx_hash        varchar         default null,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    committed_at            TIMESTAMP(0)    DEFAULT NULL,
    finalized_at            TIMESTAMP(0)    DEFAULT NULL
);

comment
on column block_batch.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';
comment
on column block_batch.rollup_status is 'undefined, pending, committing, committed, finalizing, finalized, finalization_skipped';

create unique index block_batch_id_uindex
    on block_batch (id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists  block_batch;
-- +goose StatementEnd
