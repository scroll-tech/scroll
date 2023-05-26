-- +goose Up
-- +goose StatementBegin

create table chunk_batch
(
    batch_hash              VARCHAR         NOT NULL,
    start_chunk_hash        VARCHAR         NOT NULL,
    end_chunk_hash          VARCHAR         NOT NULL,
    start_block_number      BIGINT          NOT NULL,
    start_block_hash        VARCHAR         NOT NULL,
    end_block_number        BIGINT          NOT NULL,
    end_block_hash          VARCHAR         NOT NULL,
    agg_proof               BYTEA           DEFAULT NULL,
    proving_status          INTEGER         DEFAULT 1,
    proof_time_sec          INTEGER         DEFAULT 0,
    rollup_status           INTEGER         DEFAULT 1,
    commit_tx_hash          VARCHAR         DEFAULT NULL,
    finalize_tx_hash        VARCHAR         DEFAULT NULL,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    committed_at            TIMESTAMP(0)    DEFAULT NULL,
    finalized_at            TIMESTAMP(0)    DEFAULT NULL,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (batch_hash)
);

comment
on column block_batch.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';
comment
on column block_batch.rollup_status is 'undefined, pending, committing, committed, finalizing, finalized, finalization_skipped, commit_failed, finalize_failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists chunk_batch;
-- +goose StatementEnd
