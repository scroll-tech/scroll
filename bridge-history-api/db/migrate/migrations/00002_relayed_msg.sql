-- +goose Up
-- +goose StatementBegin
create table relayed_msg
(
    id          BIGSERIAL PRIMARY KEY,
    msg_hash    VARCHAR NOT NULL,
    height      BIGINT NOT NULL,
    layer1_hash VARCHAR NOT NULL DEFAULT '',
    layer2_hash VARCHAR NOT NULL DEFAULT '',
    created_at  TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP(0) DEFAULT NULL
);

create unique index uk_msg_hash_l1_hash_l2_hash
on relayed_msg (msg_hash, layer1_hash, layer2_hash) where deleted_at IS NULL;

CREATE INDEX idx_l1_msg_relayed_msg ON relayed_msg (layer1_hash, deleted_at);

CREATE INDEX idx_l2_msg_relayed_msg ON relayed_msg (layer2_hash, deleted_at);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
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