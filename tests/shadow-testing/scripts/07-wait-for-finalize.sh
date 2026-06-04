#!/usr/bin/env bash
# Wait for on-chain finalization of bundles
# Usage: ./07-wait-for-finalize.sh --bundle-range 17297:17301 --anvil-rpc ... --scroll-chain ...

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/anvil-utils.sh"

ANVIL_RPC="${ANVIL_RPC:-http://localhost:18545}"
SCROLL_CHAIN="${SCROLL_CHAIN:-0xa13BAF47339d63B743e7Da8741db5456DAc1E556}"
DB_DSN="${DB_DSN:-postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup}"
BUNDLE_RANGE=""
TIMEOUT="${TIMEOUT:-1800}"  # 30 min default
INTERVAL="${INTERVAL:-15}"
VERIFY_ONLY=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --anvil-rpc)   ANVIL_RPC="$2"; shift 2 ;;
        --scroll-chain) SCROLL_CHAIN="$2"; shift 2 ;;
        --db-dsn)      DB_DSN="$2"; shift 2 ;;
        --bundle-range) BUNDLE_RANGE="$2"; shift 2 ;;
        --timeout)     TIMEOUT="$2"; shift 2 ;;
        --interval)    INTERVAL="$2"; shift 2 ;;
        --verify-only) VERIFY_ONLY=true; shift ;;
        -h|--help)     sed -n '2,5p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

if [[ -z "$BUNDLE_RANGE" ]]; then
    log_error "Must specify --bundle-range"
    exit 1
fi

b_start="${BUNDLE_RANGE%%:*}"
b_end="${BUNDLE_RANGE##*:}"

# We need to map bundle index to batch index.
# Query the DB to get the actual end_batch_index for the last bundle.
last_batch=$(psql "$DB_DSN" -Atq -c "
    SELECT end_batch_index FROM bundle WHERE index = $b_end
" 2>/dev/null | tr -d '\n')

if [[ -z "$last_batch" || "$last_batch" == "NULL" ]]; then
    log_warn "Could not resolve batch index for bundle $b_end, falling back to bundle index"
    last_batch="$b_end"
fi

if [[ "$VERIFY_ONLY" == "true" ]]; then
    current=$(cast call "$SCROLL_CHAIN" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$ANVIL_RPC" 2>/dev/null | awk '{print $1}')
    log_info "Current lastFinalizedBatchIndex: $current"
    log_info "Target batch index: $last_batch"
    if [[ "$current" -ge "$last_batch" ]]; then
        log_ok "All bundles finalized!"
        exit 0
    else
        log_error "Not yet finalized ($current < $last_batch)"
        exit 1
    fi
fi

log_info "Waiting for finalization..."
log_info "  Target batch: $last_batch"
log_info "  Timeout: ${TIMEOUT}s"

start_time=$(date +%s)

while true; do
    current=$(cast call "$SCROLL_CHAIN" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$ANVIL_RPC" 2>/dev/null | awk '{print $1}' || echo "0")
    elapsed=$(($(date +%s) - start_time))

    printf "\r  ⏳ lastFinalizedBatchIndex = %s / %s (%ds elapsed)" "$current" "$last_batch" "$elapsed"

    if [[ "$current" -ge "$last_batch" ]]; then
        echo ""
        log_ok "Finalization complete! lastFinalizedBatchIndex = $current"
        break
    fi

    if [[ "$elapsed" -ge "$TIMEOUT" ]]; then
        echo ""
        log_error "Timeout after ${TIMEOUT}s — lastFinalizedBatchIndex = $current"
        exit 1
    fi

    sleep "$INTERVAL"
done
