-- +goose Up
-- +goose StatementBegin
create table bridge_batch
(
    id                  BIGSERIAL PRIMARY KEY,
    batch_index         BIGINT NOT NULL,
    height              BIGINT NOT NULL,
    start_block_number  BIGINT NOT NULL,
    end_block_number    BIGINT NOT NULL,
    batch_hash          VARCHAR NOT NULL,
    status              SMALLINT DEFAULT 0,
    created_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create unique index bridge_batch_index_uindex
on bridge_batch (batch_index);

comment 
on column bridge_batch.status is 'BatchNoProof, BatchWithProof';

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON bridge_batch FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists bridge_batch;
-- +goose StatementEnd