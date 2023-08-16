-- +goose Up
-- +goose StatementBegin

create table prover_task
(
    id                  BIGSERIAL      PRIMARY KEY,

-- prover
    prover_public_key   VARCHAR        NOT NULL,
    prover_name         VARCHAR        NOT NULL,
    prover_version      VARCHAR        NOT NULL,

-- task
    task_id             VARCHAR        NOT NULL,
    task_type           SMALLINT       NOT NULL DEFAULT 0,

-- status
    proving_status      SMALLINT        NOT NULL DEFAULT 0,
    failure_type        SMALLINT        NOT NULL DEFAULT 0,
    reward              DECIMAL(78, 0)  NOT NULL DEFAULT 0,
    proof               BYTEA           DEFAULT NULL,
    assigned_at         TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,

-- metadata
    created_at          TIMESTAMP(0)   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0)   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0)   DEFAULT NULL,

    CONSTRAINT uk_tasktype_taskid_publickey UNIQUE (task_type, task_id, prover_public_key, prover_version)
);

comment
on column prover_task.task_type is 'undefined, chunk, batch';

comment
on column prover_task.proving_status is 'undefined, prover assigned, prover proof valid, prover proof invalid';

comment
on column prover_task.failure_type is 'undefined';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists prover_task;
-- +goose StatementEnd
