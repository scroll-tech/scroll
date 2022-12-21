-- +goose Up
-- +goose StatementBegin
alter table block_trace
add column message_root VARCHAR DEFAULT NULL
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table block_trace
drop column message_root
-- +goose StatementEnd