-- +goose Up
-- +goose StatementBegin

create table batch
(
    batch_index             SERIAL          NOT NULL,
    batch_hash              VARCHAR         NOT NULL,
    start_chunk_index       INTEGER         NOT NULL,
    start_chunk_hash        VARCHAR         NOT NULL,
    end_chunk_index         INTEGER         NOT NULL,
    end_chunk_hash          VARCHAR         NOT NULL,
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
);

create unique index batch_index_uindex
on batch (batch_index);

create unique index batch_hash_uindex
on batch (batch_hash);

comment
on column batch.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';

comment
on column batch.rollup_status is 'undefined, pending, committing, committed, finalizing, finalized, finalization_skipped, commit_failed, finalize_failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists batch;
-- +goose StatementEnd
