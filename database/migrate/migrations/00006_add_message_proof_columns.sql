-- +goose Up
-- +goose StatementBegin

alter table block_trace
add column message_root VARCHAR DEFAULT NULL;

alter table l1_message
add column message_proof VARCHAR DEFAULT NULL;

alter table l1_message
add column proof_height BIGINT DEFAULT 0;

create index l1_message_proof_height_index on l1_message (proof_height);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

alter table block_trace drop column message_root;

drop index l1_message_proof_height_index;

alter table l1_message drop column proof_height;

alter table l1_message drop column message_proof;

-- +goose StatementEnd