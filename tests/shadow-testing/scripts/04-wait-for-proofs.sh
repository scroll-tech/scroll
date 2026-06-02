#!/usr/bin/env bash
# Wait for bundle proofs to reach proving_status=4 (verified)
# Usage: ./04-wait-for-proofs.sh --bundle-range 17297:17301 [--timeout 3600]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/anvil-utils.sh"

DB_DSN="${DB_DSN:-postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup}"
BUNDLE_RANGE=""
TIMEOUT="${TIMEOUT:-7200}"  # Default 2 hours
INTERVAL="${INTERVAL:-30}"   # Check every 30s

while [[ $# -gt 0 ]]; do
    case "$1" in
        --db-dsn)      DB_DSN="$2"; shift 2 ;;
        --bundle-range) BUNDLE_RANGE="$2"; shift 2 ;;
        --timeout)     TIMEOUT="$2"; shift 2 ;;
        --interval)    INTERVAL="$2"; shift 2 ;;
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
total=$((b_end - b_start + 1))

log_info "Waiting for $total bundles to be proven..."
log_info "  Range: $BUNDLE_RANGE"
log_info "  Timeout: ${TIMEOUT}s"

start_time=$(date +%s)

while true; do
    result=$(psql "$DB_DSN" -Atq -c "
        SELECT COUNT(*) FROM bundle
        WHERE index BETWEEN $b_start AND $b_end
          AND proving_status = 4
    " 2>/dev/null)

    done_count=$(echo "$result" | tr -d '\n')
    elapsed=$(($(date +%s) - start_time))

    printf "\r  ⏳ %d/%d proven (%ds elapsed)" "$done_count" "$total" "$elapsed"

    if [[ "$done_count" -eq "$total" ]]; then
        echo ""
        log_ok "All $total bundles proven!"
        break
    fi

    if [[ "$elapsed" -ge "$TIMEOUT" ]]; then
        echo ""
        log_error "Timeout after ${TIMEOUT}s — only $done_count/$total proven"
        exit 1
    fi

    sleep "$INTERVAL"
done
