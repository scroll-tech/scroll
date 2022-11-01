-- +goose Up
-- +goose StatementBegin

create table block_batch
(
    id                      BIGINT          not null, -- INCREMENTAL
    -- hash                    varchar         not null, -- TODO: hash? index? id?
    total_l2_gas            BIGINT          NOT NULL DEFAULT 0,
    proving_status          integer         default 1,
    proof                   BYTEA           default null,
    instance_commitments    BYTEA           default null,
    proof_time_sec          integer         default 0,
    rollup_status           integer         default 1,
    rollup_tx_hash          varchar         default null,
    finalize_tx_hash        varchar         default null,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
drop table if exists  rollup_result;
-- +goose StatementEnd
