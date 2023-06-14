-- +goose Up
-- +goose StatementBegin
create table relayed_msg
(
    id          BIGSERIAL PRIMARY KEY,
    msg_hash    VARCHAR NOT NULL,
    height      BIGINT NOT NULL,
    layer1_hash VARCHAR NOT NULL DEFAULT '',
    layer2_hash VARCHAR NOT NULL DEFAULT '',
    is_deleted  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP(0) DEFAULT NULL
);

comment 
on column relayed_msg.is_deleted is 'NotDeleted, Deleted';

create unique index relayed_msg_hash_uindex
on relayed_msg (msg_hash);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON relayed_msg FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();

CREATE OR REPLACE FUNCTION deleted_at_trigger()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_deleted AND OLD.is_deleted != NEW.is_deleted THEN
        UPDATE relayed_msg SET deleted_at = NOW() WHERE id = NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER deleted_at_trigger
AFTER UPDATE ON relayed_msg
FOR EACH ROW
EXECUTE FUNCTION deleted_at_trigger();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists relayed_msg;
-- +goose StatementEnd