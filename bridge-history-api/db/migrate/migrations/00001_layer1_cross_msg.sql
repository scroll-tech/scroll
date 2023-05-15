-- +goose Up
-- +goose StatementBegin
create table cross_message
(
    msg_hash     VARCHAR DEFAULT '',
    height       BIGINT  NOT NULL,
    sender       VARCHAR NOT NULL,
    target       VARCHAR NOT NULL,
    amount       VARCHAR NOT NULL,
    layer1_hash  VARCHAR DEFAULT '',
    layer2_hash  VARCHAR DEFAULT '',
    layer1_token VARCHAR DEFAULT '',
    layer2_token VARCHAR DEFAULT '',
    token_id     BIGINT DEFAULT 0,
    asset        SMALLINT DEFAULT 1,
    msg_type     SMALLINT NOT NULL,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

comment
on column cross_message.asset is 'ETH, ERC20, ERC721, ERC1155, WETH';

comment
on column msg_type is 'l1msg, l2msg'

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
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
