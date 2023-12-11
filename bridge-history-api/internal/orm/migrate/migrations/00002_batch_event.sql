-- +goose Up
-- +goose StatementBegin
create table batch_event
(
    id                  BIGSERIAL     PRIMARY KEY,
    batch_status        SMALLINT      NOT NULL,
    batch_index         BIGINT        NOT NULL,
    batch_hash          VARCHAR       NOT NULL,
    start_block_number  BIGINT        NOT NULL,
    end_block_number    BIGINT        NOT NULL,
    update_status       SMALLINT      NOT NULL,
    created_at          TIMESTAMP(0)  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP(0)  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP(0)  DEFAULT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists event_batch;
-- +goose StatementEnd