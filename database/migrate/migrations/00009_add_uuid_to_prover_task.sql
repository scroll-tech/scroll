-- +goose Up
-- +goose StatementBegin
ALTER TABLE prover_task ADD COLUMN uuid uuid DEFAULT gen_random_uuid() NOT NULL UNIQUE;

DROP index uk_tasktype_taskid_publickey_version;

DROP index idx_uuid;
create index idx_uuid on prover_task (uuid) where deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index uk_tasktype_taskid_publickey_version;

create unique index uk_tasktype_taskid_publickey_version
on prover_task (task_type, task_id, prover_public_key, prover_version)) where deleted_at IS NULL;
-- +goose StatementEnd
