-- +goose Up
-- +goose StatementBegin
create table l1_message
(
    nonce        BIGINT  NOT NULL,
    height       NUMERIC NOT NULL,
    sender       VARCHAR NOT NULL,
    target       VARCHAR NOT NULL,
    value        VARCHAR NOT NULL,
    fee          VARCHAR NOT NULL,
    gas_limit    BIGINT  NOT NULL,
    deadline     BIGINT  NOT NULL,
    calldata     TEXT    NOT NULL,
    layer1_hash  VARCHAR NOT NULL,
    layer2_hash  VARCHAR DEFAULT NULL,
    status       INTEGER  DEFAULT 1,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

comment
on column l1_message.status is 'undefined, pending, submitted, confirmed';

create unique index l1_message_layer1_hash_uindex
    on l1_message (layer1_hash);

create index l1_message_height_index
    on l1_message (height);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON l1_message FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l1_message;
-- +goose StatementEnd