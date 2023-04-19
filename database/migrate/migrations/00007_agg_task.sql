-- +goose Up
-- +goose StatementBegin

create table agg_task
(
    hash         VARCHAR NOT NULL,
    task         BYTEA   NOT NULL,
    roller       TEXT DEFAULT NULL,
    proof        BYTEA DEFAULT NULL
);

create unique index agg_task_hash_uindex
    on agg_task (hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists agg_task;
-- +goose StatementEnd
