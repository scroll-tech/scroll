-- +goose Up
-- +goose StatementBegin
create table prover_sessions
(
    public_key   TEXT         NOT NULL,
    upstream     TEXT         NOT NULL,
    up_token     TEXT         NOT NULL,
    expired      TIMESTAMP(0) NOT NULL,
    constraint uk_prover_sessions_public_key_upstream unique (public_key, upstream)
);

create index idx_prover_sessions_expired on prover_sessions (expired);

create table priority_upstream
(
    public_key   TEXT         NOT NULL,
    upstream     TEXT         NOT NULL,
    update_time  TIMESTAMP(0) NOT NULL DEFAULT now()
);

create unique index idx_priority_upstream_public_key on priority_upstream (public_key);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists idx_prover_sessions_expired;

drop table if exists prover_sessions;
drop table if exists priority_upstream;
-- +goose StatementEnd