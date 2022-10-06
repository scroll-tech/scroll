-- +goose Up
-- +goose StatementBegin
create table block_result
(
    number                  integer         not null,
    hash                    varchar         not null,
    content                 json            not null,
    proof                   BYTEA           default null,
    instance_commitments    BYTEA           default null,
    status                  integer         default 1,
    tx_num                  BIGINT          NOT NULL DEFAULT 0,
    block_timestamp         NUMERIC         NOT NULL DEFAULT 0,
    proof_time_sec          integer         default 0,
    created_time            TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time            TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

comment
on column block_result.status is 'undefined, unassigned, skipped, assigned, proved, verified, failed';

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
ON block_result FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists  block_result;
-- +goose StatementEnd
