-- +goose Up
-- +goose StatementBegin
create table l2_sent_msg
(
    id               BIGSERIAL PRIMARY KEY,
    sender           VARCHAR NOT NULL,
    target           VARCHAR NOT NULL,
    value            VARCHAR NOT NULL,
    msg_hash         VARCHAR NOT NULL,
    height           BIGINT NOT NULL,
    nonce            BIGINT NOT NULL,
    batch_index      BIGINT NOT NULL DEFAULT 0,
    msg_proof        TEXT NOT NULL DEFAULT '',
    msg_data         TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at       TIMESTAMP(0) DEFAULT NULL
);

create unique index uk_msg_hash
on l2_sent_msg (msg_hash) where deleted_at IS NULL;

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_timestamp BEFORE UPDATE
ON l2_sent_msg FOR EACH ROW EXECUTE PROCEDURE
update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists l2_sent_msg;
-- +goose StatementEnd