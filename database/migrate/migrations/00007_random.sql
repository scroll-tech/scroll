-- +goose Up
-- +goose StatementBegin

create table random
(
    id                      BIGSERIAL       PRIMARY KEY,
    random                  VARCHAR         NOT NULL ,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL,
    CONSTRAINT uk_random    UNIQUE (random)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists random;
-- +goose StatementEnd
