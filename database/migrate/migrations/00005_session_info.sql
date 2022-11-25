-- +goose Up
-- +goose StatementBegin

create table session_info
(
    id           TEXT NOT NULL,
    rollers_info TEXT NOT NULL
);

create unique index session_info_id_uindex
    on session_info (id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists session_info;
-- +goose StatementEnd
