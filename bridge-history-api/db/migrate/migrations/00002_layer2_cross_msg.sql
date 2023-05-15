-- +goose Up
-- +goose StatementBegin
create table l2_cross_message
(
    msg_hash     VARCHAR DEFAULT '',
    height       BIGINT  NOT NULL,
    sender       VARCHAR NOT NULL,
    target       VARCHAR NOT NULL,
    amount       VARCHAR NOT NULL,
    layer1_hash  VARCHAR DEFAULT '',
    layer2_hash  VARCHAR NOT NULL,
    layer1_token VARCHAR DEFAULT '',
    layer2_token VARCHAR DEFAULT '',
    token_id     BIGINT DEFAULT 0,
    asset        SMALLINT  DEFAULT 1,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);
comment
on column l2_cross_message.asset is 'ETH, ERC20, ERC721, ERC1155, WETH';

create unique index l2_cross_message_hash_uindex
on l2_cross_message (layer2_hash);

create index l2_cross_message_height_index
    on l2_cross_message (height);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON l2_cross_message FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l2_cross_message;
-- +goose StatementEnd
