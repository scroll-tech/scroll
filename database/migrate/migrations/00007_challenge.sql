-- +goose Up
-- +goose StatementBegin

create table challenge
(
    id                      BIGSERIAL       PRIMARY KEY,
    challenge               VARCHAR         NOT NULL ,
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL,
    CONSTRAINT uk_challenge    UNIQUE (challenge)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists challenge;
-- +goose StatementEnd
