#!/usr/bin/env bash
# Reset shadow DB rollup_status for target bundles/batches
# Usage: ./02-prepare-db.sh [options]
#
# Options:
#   --db-dsn URL            PostgreSQL DSN (default: shadow_rollup local)
#   --bundle-range RANGE    Bundle index range, e.g. 17297:17301
#   --batch-range RANGE     Batch index range (auto-derived from bundles if omitted)
#   --no-reset-proofs       Skip resetting proving_status (only reset rollup_status)
#   --dry-run               Show SQL without executing
#   -h, --help              Show this help

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/anvil-utils.sh"

# ─── Defaults ────────────────────────────────────────────────────────────────
DB_DSN="${DB_DSN:-postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup}"
BUNDLE_RANGE=""
BATCH_RANGE=""
RESET_PROOFS=true
DRY_RUN=false

# ─── Parse args ──────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --db-dsn)         DB_DSN="$2"; shift 2 ;;
        --bundle-range)   BUNDLE_RANGE="$2"; shift 2 ;;
        --batch-range)    BATCH_RANGE="$2"; shift 2 ;;
        --no-reset-proofs) RESET_PROOFS=false; shift ;;
        --dry-run)        DRY_RUN=true; shift ;;
        -h|--help)
            sed -n '2,16p' "$0"
            exit 0
            ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

if [[ -z "$BUNDLE_RANGE" && -z "$BATCH_RANGE" ]]; then
    log_error "Must specify --bundle-range or --batch-range"
    exit 1
fi

require_cmd psql

# ─── Resolve batch range from bundles ────────────────────────────────────────
if [[ -n "$BUNDLE_RANGE" && -z "$BATCH_RANGE" ]]; then
    log_info "Resolving batch range from bundles $BUNDLE_RANGE ..."

    bundle_start="${BUNDLE_RANGE%%:*}"
    bundle_end="${BUNDLE_RANGE##*:}"

    result=$(psql "$DB_DSN" -Atq -c "
        SELECT MIN(start_batch_index), MAX(end_batch_index)
        FROM bundle
        WHERE index BETWEEN $bundle_start AND $bundle_end
    " 2>/dev/null)

    batch_start=$(echo "$result" | cut -d'|' -f1)
    batch_end=$(echo "$result" | cut -d'|' -f2)

    if [[ -z "$batch_start" || "$batch_start" == "NULL" ]]; then
        log_error "No bundles found in range $BUNDLE_RANGE"
        exit 1
    fi

    BATCH_RANGE="${batch_start}:${batch_end}"
    log_info "  Derived batch range: $BATCH_RANGE"
fi

# ─── Build SQL ───────────────────────────────────────────────────────────────
log_info "Preparing DB reset..."
log_info "  DB:    $DB_DSN"
log_info "  Bundle: ${BUNDLE_RANGE:-(n/a)}"
log_info "  Batch:  ${BATCH_RANGE:-(n/a)}"

sql_bundle=""
sql_batch=""
sql_chunk=""

if [[ -n "$BUNDLE_RANGE" ]]; then
    b_start="${BUNDLE_RANGE%%:*}"
    b_end="${BUNDLE_RANGE##*:}"
    sql_bundle="UPDATE bundle SET rollup_status = 1 WHERE index BETWEEN $b_start AND $b_end;"
    if [[ "$RESET_PROOFS" == "true" ]]; then
        sql_bundle="$sql_bundle
UPDATE bundle SET proving_status = 1, total_attempts = 0, active_attempts = 0 WHERE index BETWEEN $b_start AND $b_end;"
    fi
fi

if [[ -n "$BATCH_RANGE" ]]; then
    ba_start="${BATCH_RANGE%%:*}"
    ba_end="${BATCH_RANGE##*:}"
    sql_batch="UPDATE batch SET rollup_status = 1 WHERE index BETWEEN $ba_start AND $ba_end;"
    if [[ "$RESET_PROOFS" == "true" ]]; then
        sql_batch="$sql_batch
UPDATE batch SET proving_status = 1, total_attempts = 0, active_attempts = 0, chunk_proofs_status = 0 WHERE index BETWEEN $ba_start AND $ba_end;"
        sql_chunk="UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0 WHERE batch_hash IN (SELECT hash FROM batch WHERE index BETWEEN $ba_start AND $ba_end);"
    fi
fi

# ─── Execute or dry-run ──────────────────────────────────────────────────────
if [[ "$DRY_RUN" == "true" ]]; then
    log_info "DRY RUN — would execute:"
    echo "---"
    echo "$sql_bundle"
    echo "$sql_batch"
    echo "$sql_chunk"
    echo "---"
    exit 0
fi

log_info "Executing SQL..."

if [[ -n "$sql_bundle" ]]; then
    count=$(psql "$DB_DSN" -Atq -c "$sql_bundle" 2>/dev/null)
    log_ok "  Bundle rows updated: $count"
fi

if [[ -n "$sql_batch" ]]; then
    count=$(psql "$DB_DSN" -Atq -c "$sql_batch" 2>/dev/null)
    log_ok "  Batch rows updated: $count"
fi

if [[ -n "$sql_chunk" ]]; then
    count=$(psql "$DB_DSN" -Atq -c "$sql_chunk" 2>/dev/null)
    log_ok "  Chunk rows updated: $count"
fi

# ─── Verify ──────────────────────────────────────────────────────────────────
if [[ -n "$BUNDLE_RANGE" ]]; then
    log_info "Verifying bundle status..."
    psql "$DB_DSN" -c "
        SELECT index, proving_status, rollup_status,
               finalize_tx_hash IS NOT NULL AS has_finalize_tx
        FROM bundle
        WHERE index BETWEEN ${b_start} AND ${b_end}
        ORDER BY index
    " 2>/dev/null
fi

if [[ -n "$BATCH_RANGE" ]]; then
    log_info "Verifying batch status..."
    psql "$DB_DSN" -c "
        SELECT index, proving_status, rollup_status,
               finalize_tx_hash IS NOT NULL AS has_finalize_tx
        FROM batch
        WHERE index BETWEEN ${ba_start} AND ${ba_end}
        ORDER BY index
    " 2>/dev/null
fi

log_ok "DB preparation complete!"
