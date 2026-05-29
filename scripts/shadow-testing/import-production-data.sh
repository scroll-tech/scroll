#!/bin/bash
set -euo pipefail

# Import Production Task Data into Shadow DB
# This script exports recent batches/chunks/bundles from production RDS
# and imports them into the local shadow database.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load .env if present
if [ -f "$SCRIPT_DIR/.env" ]; then
    export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

# Build DSNs from components if not already set
PROD_DB_PASSWORD="${PROD_DB_PASSWORD:-}"
SHADOW_DB_PASSWORD="${SHADOW_DB_PASSWORD:-}"
PROD_DB="${PROD_DB:-postgresql://$PROD_DB_USER:$PROD_DB_PASSWORD@$PROD_DB_HOST:$PROD_DB_PORT/$PROD_DB_NAME}"
SHADOW_DB="${SHADOW_DB:-postgresql://$SHADOW_DB_USER:$SHADOW_DB_PASSWORD@$SHADOW_DB_HOST:$SHADOW_DB_PORT/$SHADOW_DB_NAME}"

BATCH_LIMIT="${BATCH_LIMIT:-50}"
BUNDLE_LIMIT="${BUNDLE_LIMIT:-20000}"

# Derived paths
EXPORT_DIR="${EXPORT_DIR:-/tmp/shadow-export}"
mkdir -p "$EXPORT_DIR"

log_info() {
    echo "[INFO] $1"
}

log_error() {
    echo "[ERROR] $1" >&2
}

# Verify connectivity
log_info "Checking production RDS connectivity..."
if ! psql "$PROD_DB" -c "SELECT 1;" >/dev/null 2>&1; then
    log_error "Cannot connect to production RDS at $PROD_DB"
    log_error "Ensure IDC port-forward is active (e.g., ssh -L 15432:...)"
    exit 1
fi

log_info "Checking shadow DB connectivity..."
if ! psql "$SHADOW_DB" -c "SELECT 1;" >/dev/null 2>&1; then
    log_error "Cannot connect to shadow DB at $SHADOW_DB"
    log_error "Run: ./setup.sh --postgres"
    exit 1
fi

# Get export timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
log_info "Starting export at $TIMESTAMP"

# Export batches
log_info "Exporting latest $BATCH_LIMIT batches from production..."
psql "$PROD_DB" -c "
    COPY (
        SELECT * FROM batch
        ORDER BY index DESC
        LIMIT $BATCH_LIMIT
    ) TO STDOUT WITH CSV HEADER;
" > "$EXPORT_DIR/batches_$TIMESTAMP.csv"

BATCH_COUNT=$(tail -n +2 "$EXPORT_DIR/batches_$TIMESTAMP.csv" | wc -l)
log_info "Exported $BATCH_COUNT batches"

# Get batch index range for chunk export
read -r MIN_BATCH_INDEX MAX_BATCH_INDEX <<< $(psql "$PROD_DB" -t -c "
    SELECT MIN(index), MAX(index) FROM (
        SELECT index FROM batch ORDER BY index DESC LIMIT $BATCH_LIMIT
    ) t;
" | xargs)

# Export chunks belonging to these batches
log_info "Exporting chunks for batches $MIN_BATCH_INDEX to $MAX_BATCH_INDEX..."
psql "$PROD_DB" -c "
    COPY (
        SELECT c.* FROM chunk c
        JOIN batch b ON b.start_chunk_index <= c.index AND c.index <= b.end_chunk_index
        WHERE b.index >= $MIN_BATCH_INDEX AND b.index <= $MAX_BATCH_INDEX
        ORDER BY c.index
    ) TO STDOUT WITH CSV HEADER;
" > "$EXPORT_DIR/chunks_$TIMESTAMP.csv"

CHUNK_COUNT=$(tail -n +2 "$EXPORT_DIR/chunks_$TIMESTAMP.csv" | wc -l)
log_info "Exported $CHUNK_COUNT chunks"

# Export bundles
log_info "Exporting latest $BUNDLE_LIMIT bundles..."
psql "$PROD_DB" -c "
    COPY (
        SELECT * FROM bundle
        ORDER BY index DESC
        LIMIT $BUNDLE_LIMIT
    ) TO STDOUT WITH CSV HEADER;
" > "$EXPORT_DIR/bundles_$TIMESTAMP.csv"

BUNDLE_COUNT=$(tail -n +2 "$EXPORT_DIR/bundles_$TIMESTAMP.csv" | wc -l)
log_info "Exported $BUNDLE_COUNT bundles"

# Truncate shadow tables
log_info "Clearing shadow tables..."
psql "$SHADOW_DB" -c "TRUNCATE batch, chunk, bundle CASCADE;"

# Import into shadow DB
log_info "Importing batches..."
psql "$SHADOW_DB" -c "\\copy batch FROM '$EXPORT_DIR/batches_$TIMESTAMP.csv' WITH CSV HEADER;"

log_info "Importing chunks..."
psql "$SHADOW_DB" -c "\\copy chunk FROM '$EXPORT_DIR/chunks_$TIMESTAMP.csv' WITH CSV HEADER;"

log_info "Importing bundles..."
psql "$SHADOW_DB" -c "\\copy bundle FROM '$EXPORT_DIR/bundles_$TIMESTAMP.csv' WITH CSV HEADER;"

# Reset proving status
log_info "Resetting proving status to unassigned..."
psql "$SHADOW_DB" -c "
    UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0;
    UPDATE batch SET proving_status = 1, total_attempts = 0, active_attempts = 0, chunk_proofs_status = 0;
    UPDATE bundle SET proving_status = 1, total_attempts = 0, active_attempts = 0;
"

# Summary
log_info "Import complete!"
psql "$SHADOW_DB" -c "
    SELECT 'batch' as table, COUNT(*) as cnt FROM batch
    UNION ALL SELECT 'chunk', COUNT(*) FROM chunk
    UNION ALL SELECT 'bundle', COUNT(*) FROM bundle;
"

log_info "Export files saved to: $EXPORT_DIR"
