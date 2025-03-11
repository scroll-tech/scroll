-- +goose Up
-- +goose StatementBegin

ALTER TABLE chunk
ADD COLUMN prev_l1_message_queue_hash VARCHAR DEFAULT '',
ADD COLUMN post_l1_message_queue_hash VARCHAR DEFAULT '';

ALTER TABLE batch
ADD COLUMN prev_l1_message_queue_hash VARCHAR DEFAULT '',
ADD COLUMN post_l1_message_queue_hash VARCHAR DEFAULT '';


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS chunk
DROP COLUMN IF EXISTS prev_l1_message_queue_hash,
DROP COLUMN IF EXISTS post_l1_message_queue_hash;

ALTER TABLE IF EXISTS batch
DROP COLUMN IF EXISTS prev_l1_message_queue_hash,
DROP COLUMN IF EXISTS post_l1_message_queue_hash;

-- +goose StatementEnd
