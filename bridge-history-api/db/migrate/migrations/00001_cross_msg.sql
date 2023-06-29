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
    block_timestamp TIMESTAMP(0) DEFAULT NULL,
    created_at      TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP(0) DEFAULT NULL
);

create unique index uk_layer1_hash_layer2_hash
on cross_message (layer1_hash, layer2_hash) where deleted_at IS NULL;

comment
on column cross_message.asset is 'ETH, ERC20, ERC721, ERC1155';

comment
on column cross_message.msg_type is 'unknown, l1msg, l2msg';

CREATE INDEX idx_l1_msg_index ON cross_message (layer1_hash, deleted_at);

CREATE INDEX idx_l2_msg_index ON cross_message (layer2_hash, deleted_at);

CREATE INDEX idx_height_index ON cross_message (height, msg_type, deleted_at);

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

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists cross_message;
-- +goose StatementEnd
