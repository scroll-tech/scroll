-- +goose Up
-- +goose StatementBegin

create table prover_task
(
    id                  BIGSERIAL      PRIMARY KEY,
    task_id             VARCHAR        NOT NULL,
    prover_public_key   VARCHAR        NOT NULL,
    prover_name         VARCHAR        NOT NULL,
    task_type           SMALLINT       NOT NULL DEFAULT 0,
    proving_status      SMALLINT       NOT NULL DEFAULT 0,
    failure_type        SMALLINT       NOT NULL DEFAULT 0,
    reward              BIGINT         NOT NULL DEFAULT 0,
    proof               BYTEA          DEFAULT NULL,
    created_at          TIMESTAMP(0)   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0)   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0)   DEFAULT NULL,

    CONSTRAINT uk_tasktype_taskid_publickey UNIQUE (task_type, task_id, prover_public_key)
);

comment
on column batch.proving_status is 'roller assigned, roller proof valid, roller proof invalid';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists prover_task;
-- +goose StatementEnd
