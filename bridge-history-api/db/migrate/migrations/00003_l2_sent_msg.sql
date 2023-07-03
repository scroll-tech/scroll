-- +goose Up
-- +goose StatementBegin
create table l2_sent_msg
(
    id               BIGSERIAL PRIMARY KEY,
    sender           VARCHAR NOT NULL,
    target           VARCHAR NOT NULL,
    value            VARCHAR NOT NULL,
    msg_hash         VARCHAR NOT NULL,
    height           BIGINT NOT NULL,
    nonce            BIGINT NOT NULL,
    batch_index      BIGINT NOT NULL DEFAULT 0,
    msg_proof        TEXT NOT NULL DEFAULT '',
    msg_data         TEXT NOT NULL DEFAULT '',
    is_deleted       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at       TIMESTAMP(0) DEFAULT NULL
);

comment 
on column l2_sent_msg.is_deleted is 'NotDeleted, Deleted';

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON l2_sent_msg FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();

CREATE OR REPLACE FUNCTION deleted_at_trigger()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_deleted AND OLD.is_deleted != NEW.is_deleted THEN
        UPDATE l2_sent_msg SET deleted_at = NOW() WHERE id = NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER deleted_at_trigger
AFTER UPDATE ON l2_sent_msg
FOR EACH ROW
EXECUTE FUNCTION deleted_at_trigger();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l2_sent_msg;
-- +goose StatementEnd