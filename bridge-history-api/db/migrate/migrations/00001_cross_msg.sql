-- +goose Up
-- +goose StatementBegin
create table cross_message
(
    id              BIGSERIAL PRIMARY KEY,
    msg_hash        VARCHAR NOT NULL DEFAULT '',
    height          BIGINT  NOT NULL,
    sender          VARCHAR NOT NULL,
    target          VARCHAR NOT NULL,
    amount          VARCHAR NOT NULL,
    layer1_hash     VARCHAR NOT NULL DEFAULT '',
    layer2_hash     VARCHAR NOT NULL DEFAULT '',
    layer1_token    VARCHAR NOT NULL DEFAULT '',
    layer2_token    VARCHAR NOT NULL DEFAULT '',
    token_id        BIGINT NOT NULL DEFAULT 0,
    asset           SMALLINT NOT NULL,
    msg_type        SMALLINT NOT NULL,
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
    block_timestamp TIMESTAMP(0) DEFAULT NULL,
    created_at      TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP(0) DEFAULT NULL
    CONSTRAINT      uq_layer1_hash_msg_type UNIQUE (layer1_hash, msg_type)
    CONSTRAINT      uq_layer2_hash_msg_type UNIQUE (layer2_hash, msg_type)
);

comment
on column cross_message.asset is 'ETH, ERC20, ERC721, ERC1155';

comment
on column cross_message.msg_type is 'unknown, l1msg, l2msg';

comment 
on column cross_message.is_deleted is 'NotDeleted false, Deleted true';

CREATE INDEX valid_l1_msg_index ON cross_message (layer1_hash, is_deleted);

CREATE INDEX valid_l2_msg_index ON cross_message (layer2_hash, is_deleted);

CREATE INDEX valid_height_index ON cross_message (height, msg_type, is_deleted);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON cross_message FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();

CREATE OR REPLACE FUNCTION deleted_at_trigger()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_deleted AND OLD.is_deleted != NEW.is_deleted THEN
        UPDATE cross_message SET deleted_at = NOW() WHERE id = NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER deleted_at_trigger
AFTER UPDATE ON cross_message
FOR EACH ROW
EXECUTE FUNCTION deleted_at_trigger();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists cross_message;
-- +goose StatementEnd
