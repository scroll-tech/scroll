-- +goose Up
-- +goose StatementBegin

create table session_info
(
    hash         VARCHAR NOT NULL,
    rollers_info BYTEA   NOT NULL
);

create unique index session_info_hash_uindex
    on session_info (hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists session_info;
-- +goose StatementEnd
