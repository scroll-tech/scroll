-- +goose Up
-- +goose StatementBegin
create table rollup_batch
(
    id                  BIGSERIAL PRIMARY KEY,
    batch_index         BIGINT NOT NULL,
    commit_height       BIGINT NOT NULL,
    start_block_number  BIGINT NOT NULL,
    end_block_number    BIGINT NOT NULL,
    batch_hash          VARCHAR NOT NULL,
    created_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0) DEFAULT NULL
    
);

create unique index uk_batch_index
on rollup_batch (batch_index) where deleted_at IS NULL;

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON rollup_batch FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists rollup_batch;
-- +goose StatementEnd