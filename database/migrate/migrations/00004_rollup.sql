-- +goose Up
-- +goose StatementBegin
create table rollup_result
(
    id                  BIGINT  not null, -- INCREMENTAL
    status              integer default 1,
    rollup_tx_hash      varchar default null,
    finalize_tx_hash    varchar default null,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

comment
on column rollup_result.status is 'undefined, pending, committing, committed, finalizing, finalized, finalization_skipped';

create unique index rollup_result_id_uindex
    on rollup_result (id);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON rollup_result FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists  rollup_result;
-- +goose StatementEnd
