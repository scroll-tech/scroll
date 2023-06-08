-- +goose Up
-- +goose StatementBegin

create table chunk
(
    chunk_index             SERIAL          NOT NULL,
    chunk_hash              VARCHAR         NOT NULL,
    start_block_number      BIGINT          NOT NULL,
    start_block_hash        VARCHAR         NOT NULL,
    end_block_number        BIGINT          NOT NULL,
    end_block_hash          VARCHAR         NOT NULL,
    chunk_proof             BYTEA           DEFAULT NULL,
    proof_time_sec          INTEGER         DEFAULT 0,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL,
    proving_status          INTEGER         DEFAULT 1,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    batch_index             INTEGER         DEFAULT NULL,
    batch_hash              VARCHAR         DEFAULT NULL,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
);

comment
on column chunk.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';

create unique index chunk_index_uindex
on chunk (chunk_index);

create unique index chunk_hash_uindex
on chunk (chunk_hash);

create index batch_index_index
on chunk (batch_index);

create index batch_hash_index
on chunk (batch_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists chunk;
-- +goose StatementEnd
