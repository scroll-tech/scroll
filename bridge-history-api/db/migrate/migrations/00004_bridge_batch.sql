-- +goose Up
-- +goose StatementBegin
create table bridge_batch
(
    id                  BIGSERIAL PRIMARY KEY,
    height              BIGINT NOT NULL,
    start_block_number  BIGINT NOT NULL,
    end_block_number    BIGINT NOT NULL,
    created_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
);

comment 
on column bridge_batch.is_deleted is 'NotDeleted, Deleted';

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