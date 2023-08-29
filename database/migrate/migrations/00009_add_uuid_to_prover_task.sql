-- +goose Up
-- +goose StatementBegin
ALTER TABLE prover_task ADD COLUMN uuid uuid DEFAULT gen_random_uuid() NOT NULL UNIQUE;

create index if not exists idx_uuid on prover_task (uuid) where deleted_at IS NULL;

ALTER TABLE prover_task DROP CONSTRAINT uk_tasktype_taskid_publickey_version;

drop index if exists uk_tasktype_taskid_publickey_version;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
create unique index if not exists uk_tasktype_taskid_publickey_version
    on prover_task (task_type, task_id, prover_public_key, prover_version) where deleted_at IS NULL;
-- +goose StatementEnd
