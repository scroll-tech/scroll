-- +goose Up
-- +goose StatementBegin

create table agg_session_info
(
    hash         VARCHAR NOT NULL,
    rollers_info BYTEA   NOT NULL
);

create unique index agg_session_info_hash_uindex
    on agg_session_info (hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists agg_session_info;
-- +goose StatementEnd
