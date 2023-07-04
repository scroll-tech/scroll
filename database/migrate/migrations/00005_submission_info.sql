-- +goose Up
-- +goose StatementBegin

create table submission_info
(
    id                  BIGSERIAL      PRIMARY KEY,
    task_id             VARCHAR        NOT NULL,
    roller_public_key   VARCHAR        NOT NULL,
    prove_type          SMALLINT       DEFAULT 0,
    roller_name         VARCHAR        NOT NULL,
    proving_status      SMALLINT       DEFAULT 1,
    failure_type        SMALLINT       DEFAULT 0,
    reward              BIGINT         DEFAULT 0,
    proof               BYTEA          DEFAULT NULL,
    created_at          TIMESTAMP(0)   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0)   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0)   DEFAULT NULL,

    CONSTRAINT uk_session_unique UNIQUE (task_id, roller_public_key)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists submission_info;
-- +goose StatementEnd
