-- +goose Up
-- +goose StatementBegin
ALTER TABLE batch ADD PRIMARY KEY (index);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE batch DROP CONSTRAINT batch_pkey;
-- +goose StatementEnd