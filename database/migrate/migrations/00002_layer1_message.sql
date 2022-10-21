-- +goose Up
-- +goose StatementBegin
create table layer1_message
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
    layer1_hash varchar not null,
    layer2_hash varchar default null,
    status      integer  default 1,
    created_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

comment
on column layer1_message.status is 'undefined, pending, submitted, confirmed';

create unique index layer1_message_nonce_uindex
    on layer1_message (nonce);

create index layer1_message_height_index
    on layer1_message (height);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_time = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON layer1_message FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists layer1_message;
-- +goose StatementEnd