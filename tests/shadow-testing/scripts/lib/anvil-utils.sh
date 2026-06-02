#!/usr/bin/env bash
# Shared Anvil utility functions for shadow testing
set -euo pipefail

# Note: do not define SCRIPT_DIR here, as this file is sourced by other scripts
# and would overwrite their SCRIPT_DIR variable.

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }

# Wait for Anvil to be ready
wait_for_anvil() {
    local rpc_url="${1:-http://localhost:18545}"
    local max_wait="${2:-60}"
    log_info "Waiting for Anvil at $rpc_url ..."
    for ((i=0; i<max_wait; i++)); do
        if cast block-number --rpc-url "$rpc_url" >/dev/null 2>&1; then
            log_ok "Anvil is ready"
            return 0
        fi
        sleep 1
    done
    log_error "Anvil did not become ready within ${max_wait}s"
    return 1
}

# Get storage slot value
get_storage() {
    local contract="$1"
    local slot="$2"
    local rpc_url="${3:-http://localhost:18545}"
    cast storage "$contract" "$slot" --rpc-url "$rpc_url" 2>/dev/null | tr -d '\n'
}

# Set storage slot value (Anvil only)
set_storage() {
    local contract="$1"
    local slot="$2"
    local value="$3"
    local rpc_url="${4:-http://localhost:18545}"
    cast rpc anvil_setStorageAt "$contract" "$slot" "$value" --rpc-url "$rpc_url" >/dev/null
}

# Impersonate an account (Anvil only)
impersonate() {
    local addr="$1"
    local rpc_url="${2:-http://localhost:18545}"
    cast rpc anvil_impersonateAccount "$addr" --rpc-url "$rpc_url" >/dev/null
}

# Stop impersonating
stop_impersonate() {
    local addr="$1"
    local rpc_url="${2:-http://localhost:18545}"
    cast rpc anvil_stopImpersonatingAccount "$addr" --rpc-url "$rpc_url" >/dev/null
}

# Send ETH to an address
set_balance() {
    local addr="$1"
    local wei="${2:-0x56bc75e2d63100000}"  # 100 ETH default
    local rpc_url="${3:-http://localhost:18545}"
    cast rpc anvil_setBalance "$addr" "$wei" --rpc-url "$rpc_url" >/dev/null
}

# Encode uint256 for storage
encode_uint256() {
    local val="$1"
    printf '%064x' "$val"
}

# Extract lower 64 bits from a 32-byte hex string
lower64() {
    local hex="$1"
    echo "${hex: -16}"
}

# Check if a command exists
require_cmd() {
    if ! command -v "$1" &>/dev/null; then
        log_error "Required command not found: $1"
        exit 1
    fi
}

# Parse bundle range like "17297:17301" into individual indices
parse_bundle_range() {
    local range="$1"
    local start="${range%%:*}"
    local end="${range##*:}"
    for ((i=start; i<=end; i++)); do
        echo "$i"
    done
}
