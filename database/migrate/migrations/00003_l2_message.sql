-- +goose Up
-- +goose StatementBegin
create table l2_message
(
    nonce        BIGINT  NOT NULL,
    msg_hash     VARCHAR NOT NULL,
    height       BIGINT  NOT NULL,
    sender       VARCHAR NOT NULL,
    target       VARCHAR NOT NULL,
    value        VARCHAR NOT NULL,
    calldata     TEXT    NOT NULL,
    layer2_hash  VARCHAR NOT NULL,
    layer1_hash  VARCHAR DEFAULT NULL,
    proof        TEXT    DEFAULT NULL,
    status       INTEGER  DEFAULT 1,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

comment
on column l2_message.status is 'undefined, pending, submitted, confirmed, failed, expired, relay_failed';

create unique index l2_message_hash_uindex
on l2_message (msg_hash);

create unique index l2_message_nonce_uindex
on l2_message (nonce);

create index l2_message_height_index
    on l2_message (height);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON l2_message FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l2_message;
-- +goose StatementEnd