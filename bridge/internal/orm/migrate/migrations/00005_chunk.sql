-- +goose Up
-- +goose StatementBegin

create table chunk
(
    chunk_hash              VARCHAR         NOT NULL,
    start_block_number      BIGINT          NOT NULL,
    start_block_hash        VARCHAR         NOT NULL,
    end_block_number        BIGINT          NOT NULL,
    end_block_hash          VARCHAR         NOT NULL,
    block_contexts          TEXT            NOT NULL,
    zkevm_proof             BYTEA           DEFAULT NULL,
    proof_time_sec          INTEGER         DEFAULT 0,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    primary key (chunk_hash)
);

comment
on column chunk.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists chunk;
-- +goose StatementEnd
