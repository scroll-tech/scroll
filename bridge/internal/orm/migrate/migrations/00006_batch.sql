-- +goose Up
-- +goose StatementBegin

create table batch
(
    index                   BIGINT          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    start_chunk_index       INTEGER         NOT NULL,
    start_chunk_hash        VARCHAR         NOT NULL,
    end_chunk_index         INTEGER         NOT NULL,
    end_chunk_hash          VARCHAR         NOT NULL,
    batch_header            BYTEA           DEFAULT NULL,
    state_root              VARCHAR         DEFAULT NULL,
    withdraw_root           VARCHAR         DEFAULT NULL,
    proof                   BYTEA           DEFAULT NULL,
    proving_status          INTEGER         NOT NULL DEFAULT 1,
    proof_time_sec          INTEGER         DEFAULT NULL,
    rollup_status           INTEGER         NOT NULL DEFAULT 1,
    commit_tx_hash          VARCHAR         DEFAULT NULL,
    finalize_tx_hash        VARCHAR         DEFAULT NULL,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    committed_at            TIMESTAMP(0)    DEFAULT NULL,
    finalized_at            TIMESTAMP(0)    DEFAULT NULL,
    oracle_status           INTEGER         NOT NULL DEFAULT 1,
    oracle_tx_hash          VARCHAR         DEFAULT NULL,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL
);

create unique index batch_index_uindex
on batch (index);

create unique index batch_hash_uindex
on batch (hash);

comment
on column batch.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';

comment
on column batch.rollup_status is 'undefined, pending, committing, committed, finalizing, finalized, finalization_skipped, commit_failed, finalize_failed';

create or replace function update_timestamp()
returns trigger as $$
begin
   NEW.updated_at = current_timestamp;
   return NEW;
end;
$$ language 'plpgsql';

create trigger update_timestamp before update
on batch for each row execute procedure
update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists batch;
-- +goose StatementEnd
