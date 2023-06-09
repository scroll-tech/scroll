-- +goose Up
-- +goose StatementBegin

create table chunk
(
    index                   SERIAL          NOT NULL,
    hash                    VARCHAR         NOT NULL,
    start_block_number      BIGINT          NOT NULL,
    start_block_hash        VARCHAR         NOT NULL,
    end_block_number        BIGINT          NOT NULL,
    end_block_hash          VARCHAR         NOT NULL,
    chunk_proof             BYTEA           DEFAULT NULL,
    proof_time_sec          INTEGER         DEFAULT NULL,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL,
    proving_status          INTEGER         NOT NULL DEFAULT 1,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    batch_index             INTEGER         DEFAULT NULL,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL,
);

comment
on column chunk.proving_status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';

create unique index chunk_index_uindex
on chunk (index);

create unique index chunk_hash_uindex
on chunk (hash);

create index batch_index_index
on chunk (batch_index);

create or replace function update_timestamp()
returns trigger as $$
begin
   NEW.updated_at = current_timestamp;
   return NEW;
end;
$$ language 'plpgsql';

create trigger update_timestamp before update
on chunk for each row execute procedure
update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists chunk;
-- +goose StatementEnd
