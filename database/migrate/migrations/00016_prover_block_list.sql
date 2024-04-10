-- +goose Up
-- +goose StatementBegin

CREATE TABLE prover_block_list
(
    id           BIGSERIAL    PRIMARY KEY,

    public_key   VARCHAR      NOT NULL,

-- debug info
    prover_name  VARCHAR      NOT NULL,

    created_at   TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at   TIMESTAMP(0) DEFAULT NULL
);

CREATE INDEX idx_prover_block_list_on_public_key ON prover_block_list(public_key);
CREATE INDEX idx_prover_block_list_on_prover_name ON prover_block_list(prover_name);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS prover_block_list;
-- +goose StatementEnd
