-- +goose Up
-- +goose StatementBegin
create table relayed_msg
(
    msg_hash     VARCHAR NOT NULL,
    height       BIGINT NOT NULL,
    layer1_hash  VARCHAR DEFAULT '',
    layer2_hash  VARCHAR DEFAULT '',
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create unique index relayed_msg_hash_uindex
on relayed_msg (msg_hash);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON relayed_msg FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists relayed_msg;
-- +goose StatementEnd