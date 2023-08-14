-- +goose Up
-- +goose StatementBegin

create table batch
(
-- batch
    index                   BIGINT          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    start_chunk_index       BIGINT          NOT NULL,
    start_chunk_hash        VARCHAR         NOT NULL,
    end_chunk_index         BIGINT          NOT NULL,
    end_chunk_hash          VARCHAR         NOT NULL,
    state_root              VARCHAR         NOT NULL,
    withdraw_root           VARCHAR         NOT NULL,
    parent_batch_hash       VARCHAR         NOT NULL,
    batch_header            BYTEA           NOT NULL,

-- proof
    chunk_proofs_status     SMALLINT        NOT NULL DEFAULT 1,
    proving_status          SMALLINT        NOT NULL DEFAULT 1,
    proof                   BYTEA           DEFAULT NULL,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL, -- DEPRECATED
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    proof_time_sec          INTEGER         DEFAULT NULL,

-- rollup
    rollup_status           SMALLINT        NOT NULL DEFAULT 1,
    commit_tx_hash          VARCHAR         DEFAULT NULL,
    committed_at            TIMESTAMP(0)    DEFAULT NULL,
    finalize_tx_hash        VARCHAR         DEFAULT NULL,
    finalized_at            TIMESTAMP(0)    DEFAULT NULL,

-- gas oracle
    oracle_status           SMALLINT        NOT NULL DEFAULT 1,
    oracle_tx_hash          VARCHAR         DEFAULT NULL,

-- metadata
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL
);

create unique index batch_index_uindex
on batch (index) where deleted_at IS NULL;

create unique index batch_hash_uindex
on batch (hash) where deleted_at IS NULL;

comment
on column batch.chunk_proofs_status is 'undefined, pending, ready';

comment
on column batch.proving_status is 'undefined, unassigned, assigned, proved, verified, failed';

comment
on column batch.rollup_status is 'undefined, pending, committing, committed, finalizing, finalized, commit_failed, finalize_failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists batch;
-- +goose StatementEnd
