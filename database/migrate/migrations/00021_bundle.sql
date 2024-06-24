-- +goose Up
-- +goose StatementBegin

CREATE TABLE bundle (
    index                   BIGSERIAL      PRIMARY KEY,
    start_batch_index       BIGINT          NOT NULL,
    end_batch_index         BIGINT          NOT NULL,
    start_batch_hash        VARCHAR         NOT NULL,
    end_batch_hash          VARCHAR         NOT NULL,

-- proof
    batch_proofs_status     SMALLINT        NOT NULL DEFAULT 1,
    proving_status          SMALLINT        NOT NULL DEFAULT 1,
    proof                   BYTEA           DEFAULT NULL,
    prover_assigned_at      TIMESTAMP(0)    DEFAULT NULL,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    proof_time_sec          INTEGER         DEFAULT NULL,

-- rollup
    rollup_status           SMALLINT        NOT NULL DEFAULT 1,
    finalize_tx_hash        VARCHAR         DEFAULT NULL,
    finalized_at            TIMESTAMP(0)    DEFAULT NULL,

-- metadata
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL
);

CREATE INDEX bundle_start_batch_index_idx ON bundle (start_batch_index) WHERE deleted_at IS NULL;
CREATE INDEX bundle_end_batch_index_idx ON bundle (end_batch_index) WHERE deleted_at IS NULL;

COMMENT ON COLUMN bundle.batch_proofs_status IS 'undefined, pending, ready';
COMMENT ON COLUMN bundle.proving_status IS 'undefined, unassigned, assigned, proved (deprecated), verified, failed';
COMMENT ON COLUMN bundle.rollup_status IS 'undefined, pending, committing (not used for bundles), committed (not used for bundles), finalizing, finalized, commit_failed (not used for bundles), finalize_failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bundle;
-- +goose StatementEnd
