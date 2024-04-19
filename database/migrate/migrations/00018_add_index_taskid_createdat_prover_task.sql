-- +goose Up
-- +goose StatementBegin

create index if not exists idx_prover_task_created_at on prover_task(created_at) where deleted_at IS NULL;

create index if not exists idx_prover_task_task_id on prover_task(task_id) where deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop index if exists idx_prover_task_created_at;

drop index if exists idx_prover_task_task_id;


-- +goose StatementEnd
