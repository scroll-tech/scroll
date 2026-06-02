#!/bin/bash
# Import a specific bundle range from production RDS into the shadow DB.
# Usage: ./00-import-bundle-range.sh [options]
#
# Options:
#   --bundle-range RANGE    Bundle index range, e.g. 17302:17305
#   --prod-dsn DSN          Production RDS connection string
#   --shadow-dsn DSN        Shadow DB connection string
#   --dry-run               Show SQL without executing
#   -h, --help              Show this help

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/anvil-utils.sh"

# ─── Defaults ────────────────────────────────────────────────────────────────
PROD_DSN="${PROD_DSN:-postgresql://postgres:postgres@localhost:15432/rollup}"
SHADOW_DSN="${SHADOW_DSN:-postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup}"
BUNDLE_RANGE=""
DRY_RUN=false

# ─── Parse args ──────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --bundle-range)   BUNDLE_RANGE="$2"; shift 2 ;;
        --prod-dsn)       PROD_DSN="$2"; shift 2 ;;
        --shadow-dsn)     SHADOW_DSN="$2"; shift 2 ;;
        --dry-run)        DRY_RUN=true; shift ;;
        -h|--help)
            sed -n '2,12p' "$0"
            exit 0
            ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

if [[ -z "$BUNDLE_RANGE" ]]; then
    log_error "Must specify --bundle-range (e.g., 17302:17305)"
    exit 1
fi

BUNDLE_START="${BUNDLE_RANGE%%:*}"
BUNDLE_END="${BUNDLE_RANGE##*:}"

require_cmd psql

# ─── Verify connectivity ─────────────────────────────────────────────────────
log_info "Checking production RDS connectivity..."
if ! psql "$PROD_DSN" -c "SELECT 1;" >/dev/null 2>&1; then
    log_error "Cannot connect to production RDS at $PROD_DSN"
    log_error "Ensure IDC port-forward is active (e.g., ssh -L 15432:...:5432 idc-us-1-19)"
    exit 1
fi

log_info "Checking shadow DB connectivity..."
if ! psql "$SHADOW_DSN" -c "SELECT 1;" >/dev/null 2>&1; then
    log_error "Cannot connect to shadow DB at $SHADOW_DSN"
    log_error "Run: docker compose up postgres -d"
    exit 1
fi

# ─── Resolve batch range from bundles ────────────────────────────────────────
log_info "Resolving batch range from bundles $BUNDLE_RANGE ..."

RANGE_SQL="
SELECT MIN(start_batch_index), MAX(end_batch_index)
FROM bundle
WHERE index BETWEEN $BUNDLE_START AND $BUNDLE_END
"

result=$(psql "$PROD_DSN" -Atq -c "$RANGE_SQL" 2>/dev/null | xargs)
BATCH_START=$(echo "$result" | cut -d'|' -f1 | tr -d ' ')
BATCH_END=$(echo "$result" | cut -d'|' -f2 | tr -d ' ')

if [[ -z "$BATCH_START" || "$BATCH_START" == "NULL" ]]; then
    log_error "No bundles found in range $BUNDLE_RANGE on production RDS"
    exit 1
fi

log_info "  Bundle range: $BUNDLE_START → $BUNDLE_END"
log_info "  Batch range:  $BATCH_START → $BATCH_END"

# ─── Export dir ──────────────────────────────────────────────────────────────
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
EXPORT_DIR="/tmp/shadow-export-$TIMESTAMP"
mkdir -p "$EXPORT_DIR"

# ─── Export from production RDS ──────────────────────────────────────────────
log_info "Exporting bundles $BUNDLE_START..$BUNDLE_END from production..."
psql "$PROD_DSN" -c "
    COPY (
        SELECT * FROM bundle
        WHERE index BETWEEN $BUNDLE_START AND $BUNDLE_END
        ORDER BY index
    ) TO STDOUT WITH CSV HEADER;
" > "$EXPORT_DIR/bundles.csv"

BUNDLE_COUNT=$(tail -n +2 "$EXPORT_DIR/bundles.csv" | wc -l)
log_info "  Exported $BUNDLE_COUNT bundles"

log_info "Exporting batches $BATCH_START..$BATCH_END from production..."
psql "$PROD_DSN" -c "
    COPY (
        SELECT * FROM batch
        WHERE index BETWEEN $BATCH_START AND $BATCH_END
        ORDER BY index
    ) TO STDOUT WITH CSV HEADER;
" > "$EXPORT_DIR/batches.csv"

BATCH_COUNT=$(tail -n +2 "$EXPORT_DIR/batches.csv" | wc -l)
log_info "  Exported $BATCH_COUNT batches"

log_info "Exporting chunks for batches $BATCH_START..$BATCH_END..."
psql "$PROD_DSN" -c "
    COPY (
        SELECT c.* FROM chunk c
        JOIN batch b ON b.start_chunk_index <= c.index AND c.index <= b.end_chunk_index
        WHERE b.index BETWEEN $BATCH_START AND $BATCH_END
        ORDER BY c.index
    ) TO STDOUT WITH CSV HEADER;
" > "$EXPORT_DIR/chunks.csv"

CHUNK_COUNT=$(tail -n +2 "$EXPORT_DIR/chunks.csv" | wc -l)
log_info "  Exported $CHUNK_COUNT chunks"

log_info "Exporting l2_blocks for chunks..."
psql "$PROD_DSN" -c "
    COPY (
        SELECT l.* FROM l2_block l
        JOIN chunk c ON c.hash = l.chunk_hash
        JOIN batch b ON b.start_chunk_index <= c.index AND c.index <= b.end_chunk_index
        WHERE b.index BETWEEN $BATCH_START AND $BATCH_END
        ORDER BY l.number
    ) TO STDOUT WITH CSV HEADER;
" > "$EXPORT_DIR/l2_blocks.csv"

L2BLOCK_COUNT=$(tail -n +2 "$EXPORT_DIR/l2_blocks.csv" | wc -l)
log_info "  Exported $L2BLOCK_COUNT l2_blocks"

# ─── Check for parent batch ────────────────────────────────────────────────────
log_info "Checking parent batch (batch $((BATCH_START - 1)))..."
PARENT_EXISTS=$(psql "$PROD_DSN" -Atq -c "
    SELECT COUNT(*) FROM batch WHERE index = $((BATCH_START - 1))
" 2>/dev/null | tr -d ' ')

if [[ "$PARENT_EXISTS" == "0" ]]; then
    log_warn "  Parent batch $((BATCH_START - 1)) not found in production"
    log_warn "  Coordinator bundle task generation will fail without parent batch"
else
    log_info "  Exporting parent batch $((BATCH_START - 1))..."
    psql "$PROD_DSN" -c "
        COPY (
            SELECT * FROM batch WHERE index = $((BATCH_START - 1))
        ) TO STDOUT WITH CSV HEADER;
    " > "$EXPORT_DIR/parent_batch.csv"
fi

# ─── Truncate shadow tables ──────────────────────────────────────────────────
log_info "Clearing shadow tables..."
if [[ "$DRY_RUN" == "true" ]]; then
    log_info "DRY RUN — would execute: TRUNCATE batch, chunk, bundle, l2_block CASCADE;"
else
    psql "$SHADOW_DSN" -c "TRUNCATE batch, chunk, bundle, l2_block CASCADE;" >/dev/null
fi

# ─── Import into shadow DB ───────────────────────────────────────────────────
log_info "Importing into shadow DB..."

import_csv() {
    local table="$1"
    local file="$2"
    if [[ -f "$file" ]]; then
        local count=$(tail -n +2 "$file" | wc -l)
        if [[ "$count" -gt 0 ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "  DRY RUN: would import $count rows into $table"
            else
                psql "$SHADOW_DSN" -c "\copy $table FROM '$file' WITH CSV HEADER;" >/dev/null
                log_ok "  Imported $count rows into $table"
            fi
        fi
    fi
}

import_csv "bundle" "$EXPORT_DIR/bundles.csv"
import_csv "batch" "$EXPORT_DIR/batches.csv"
import_csv "chunk" "$EXPORT_DIR/chunks.csv"
import_csv "l2_block" "$EXPORT_DIR/l2_blocks.csv"

if [[ -f "$EXPORT_DIR/parent_batch.csv" ]]; then
    import_csv "batch" "$EXPORT_DIR/parent_batch.csv"
fi

# ─── Reset status ────────────────────────────────────────────────────────────
log_info "Resetting proving & rollup status..."
if [[ "$DRY_RUN" == "true" ]]; then
    log_info "  DRY RUN: would reset proving_status and rollup_status"
else
    psql "$SHADOW_DSN" -c "
        UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0;
        UPDATE batch SET proving_status = 1, total_attempts = 0, active_attempts = 0, chunk_proofs_status = 0;
        UPDATE bundle SET proving_status = 1, total_attempts = 0, active_attempts = 0, rollup_status = 1;
    " >/dev/null
    log_ok "  Status reset complete"
fi

# ─── Verify ──────────────────────────────────────────────────────────────────
log_info "Verifying shadow DB..."
psql "$SHADOW_DSN" -c "
    SELECT 'batch' as table, COUNT(*) as cnt FROM batch
    UNION ALL SELECT 'chunk', COUNT(*) FROM chunk
    UNION ALL SELECT 'bundle', COUNT(*) FROM bundle;
" 2>/dev/null

log_ok "Import complete! Export files saved to: $EXPORT_DIR"
