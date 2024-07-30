-- +goose Up
-- +goose StatementBegin

CREATE TABLE bundle (
    index                   BIGSERIAL       PRIMARY KEY,
    hash                    VARCHAR         NOT NULL, -- Not part of DA hash, used for SQL query consistency and ease of use, derived using keccak256(concat(start_batch_hash_bytes, end_batch_hash_bytes)).
    start_batch_index       BIGINT          NOT NULL,
    end_batch_index         BIGINT          NOT NULL,
    start_batch_hash        VARCHAR         NOT NULL,
    end_batch_hash          VARCHAR         NOT NULL,
    codec_version           SMALLINT        NOT NULL,

-- proof
    batch_proofs_status     SMALLINT        NOT NULL DEFAULT 1,
    proving_status          SMALLINT        NOT NULL DEFAULT 1,
    proof                   BYTEA           DEFAULT NULL,
    proved_at               TIMESTAMP(0)    DEFAULT NULL,
    proof_time_sec          INTEGER         DEFAULT NULL,
    total_attempts          SMALLINT        NOT NULL DEFAULT 0,
    active_attempts         SMALLINT        NOT NULL DEFAULT 0,

-- rollup
    rollup_status           SMALLINT        NOT NULL DEFAULT 1,
    finalize_tx_hash        VARCHAR         DEFAULT NULL,
    finalized_at            TIMESTAMP(0)    DEFAULT NULL,

-- metadata
    created_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP(0)    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMP(0)    DEFAULT NULL
);

CREATE INDEX idx_bundle_index_rollup_status ON bundle(index, rollup_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_bundle_hash ON bundle(hash) WHERE deleted_at IS NULL;
CREATE INDEX idx_bundle_hash_proving_status ON bundle(hash, proving_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_bundle_index_desc ON bundle(index DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_bundle_batch_proofs_status ON bundle(batch_proofs_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_bundle_start_batch_index ON bundle(start_batch_index) WHERE deleted_at IS NULL;
CREATE INDEX idx_bundle_end_batch_index ON bundle(end_batch_index) WHERE deleted_at IS NULL;
create index idx_bundle_total_attempts_active_attempts_batch_proofs_status
    on bundle (total_attempts, active_attempts, batch_proofs_status)
    where deleted_at IS NULL;

COMMENT ON COLUMN bundle.batch_proofs_status IS 'undefined, pending, ready';
COMMENT ON COLUMN bundle.proving_status IS 'undefined, unassigned, assigned, proved (deprecated), verified, failed';
COMMENT ON COLUMN bundle.rollup_status IS 'undefined, pending, committing (not used for bundles), committed (not used for bundles), finalizing, finalized, commit_failed (not used for bundles), finalize_failed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bundle;
-- +goose StatementEnd
