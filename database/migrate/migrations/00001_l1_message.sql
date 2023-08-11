-- +goose Up
-- +goose StatementBegin
create table l1_message
(
    queue_index      BIGINT           NOT NULL,
    msg_hash         VARCHAR          NOT NULL,
    height           BIGINT           NOT NULL,
    gas_limit        BIGINT           NOT NULL,
    sender           VARCHAR          NOT NULL,
    target           VARCHAR          NOT NULL,
    value            VARCHAR          NOT NULL,
    calldata         TEXT             NOT NULL,
    layer1_hash      VARCHAR          NOT NULL,
    layer2_hash      VARCHAR          DEFAULT NULL,
    status           INTEGER          NOT NULL DEFAULT 1,

-- metadata
    created_at       TIMESTAMP(0)     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP(0)     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at       TIMESTAMP(0)     DEFAULT NULL
);

comment
on column l1_message.status is 'undefined, pending, submitted, confirmed, failed, expired, relay_failed';

create unique index l1_message_hash_uindex
on l1_message (msg_hash) where deleted_at IS NULL;

create unique index l1_message_nonce_uindex
on l1_message (queue_index) where deleted_at IS NULL;

create index l1_message_height_index
on l1_message (height) where deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l1_message;
-- +goose StatementEnd
