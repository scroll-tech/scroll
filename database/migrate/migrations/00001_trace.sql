-- +goose Up
-- +goose StatementBegin
create table prove_task
(
    -- hash                    varchar         not null, -- TODO: hash? index? id?
    id                      BIGINT          not null, -- INCREMENTAL
    total_l2_gas            BIGINT          NOT NULL DEFAULT 0,
    proof                   BYTEA           default null,
    instance_commitments    BYTEA           default null,
    status                  integer         default 1,
    proof_time_sec          integer         default 0,
    created_time            TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time            TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create table block_result
(
    number                  BIGINT          not null,
    hash                    varchar         not null,
    content                 json            not null,
    task_id                 integer         default null, -- TODO: foreign key?
    tx_num                  integer         NOT NULL DEFAULT 0, -- FIXME: why tx_num is bigint?
    gas_used                BIGINT          NOT NULL DEFAULT 0,
    block_timestamp         NUMERIC         NOT NULL DEFAULT 0,
);

comment
on column prove_task.status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';

create unique index prove_task_id_uindex
    on prove_task (id);

create unique index block_result_hash_uindex
    on block_result (hash);

create unique index block_result_number_uindex
    on block_result (number);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON prove_task FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists  block_result;
drop table if exists  prove_task;
-- +goose StatementEnd
