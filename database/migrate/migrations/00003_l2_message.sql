-- +goose Up
-- +goose StatementBegin
create table l2_message
(
    nonce       bigint  not null,
    height      bigint  not null,
    sender      varchar not null,
    target      varchar not null,
    value       varchar not null,
    fee         varchar not null,
    gas_limit   bigint  not null,
    deadline    bigint  not null,
    calldata    text    not null,
    layer2_hash varchar not null,
    layer1_hash varchar default null,
    proof       text    default null,
    status      integer  default 1,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

comment
on column l2_message.status is 'undefined, pending, submitted, confirmed';

create unique index l2_message_layer2_hash_uindex
    on l2_message (layer2_hash);

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