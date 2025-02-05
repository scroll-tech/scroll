-- +goose Up
-- +goose StatementBegin

ALTER TABLE chunk
ADD COLUMN initial_l1_message_queue_hash VARCHAR DEFAULT '';
ADD COLUMN last_l1_message_queue_hash VARCHAR DEFAULT '';

ALTER TABLE batch
ADD COLUMN initial_l1_message_queue_hash VARCHAR DEFAULT '';
ADD COLUMN last_l1_message_queue_hash VARCHAR DEFAULT '';


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE IF EXISTS chunk
DROP COLUMN IF EXISTS initial_l1_message_queue_hash;
DROP COLUMN IF EXISTS last_l1_message_queue_hash;

ALTER TABLE IF EXISTS batch
DROP COLUMN IF EXISTS initial_l1_message_queue_hash;
DROP COLUMN IF EXISTS last_l1_message_queue_hash;

-- +goose StatementEnd
