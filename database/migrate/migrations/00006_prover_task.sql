-- +goose Up
-- +goose StatementBegin

create table prover_task
(
    id                  BIGSERIAL      PRIMARY KEY,
    task_id             VARCHAR        NOT NULL,
    roller_public_key   VARCHAR        NOT NULL,
    roller_name         VARCHAR        NOT NULL,
    proof_type          SMALLINT       NOT NULL DEFAULT 0,
    proving_status      SMALLINT       NOT NULL DEFAULT 0,
    failure_type        SMALLINT       NOT NULL DEFAULT 0,
    reward              BIGINT         NOT NULL DEFAULT 0,
    proof               BYTEA          DEFAULT NULL,
    created_at          TIMESTAMP(0)   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0)   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0)   DEFAULT NULL,

    CONSTRAINT uk_taskid_publickey UNIQUE (task_id, roller_public_key)
);

comment
on column batch.proving_status is 'roller assigned, roller proof valid, roller proof invalid';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists prover_task;
-- +goose StatementEnd
