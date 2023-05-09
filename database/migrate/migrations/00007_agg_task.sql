-- +goose Up
-- +goose StatementBegin

create table agg_task
(
    id                     VARCHAR NOT NULL,
    start_batch_index      BIGINT  NOT NULL,
    start_batch_hash       VARCHAR  NOT NULL,
    end_batch_index        BIGINT   NOT NULL,
    end_batch_hash         VARCHAR  NOT NULL,
    proving_status         SMALLINT DEFAULT 1,
    proof                  BYTEA DEFAULT NULL,
    created_time           TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time           TIMESTAMP(0)    DEFAULT CURRENT_TIMESTAMP
);

create unique index agg_task_hash_uindex
    on agg_task (id);


CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON agg_task FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists agg_task;
-- +goose StatementEnd
