#!/usr/bin/env bash
# Build and launch the rollup relayer
# Usage: ./05-run-relayer.sh --config mainnet [--build]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/anvil-utils.sh"

CONFIG="${CONFIG:-mainnet}"
BUILD=false
DOCKER=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)      CONFIG="$2"; shift 2 ;;
        --build)       BUILD=true; shift ;;
        --docker)      DOCKER=true; shift ;;
        -h|--help)     sed -n '2,5p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

CONFIG_FILE="${SCRIPT_DIR}/../configs/${CONFIG}.json"
if [[ ! -f "$CONFIG_FILE" ]]; then
    log_error "Config not found: $CONFIG_FILE"
    exit 1
fi

# Read values from config
ANVIL_RPC=$(jq -r '.fork.anvil_rpc' "$CONFIG_FILE")
DB_DSN=$(jq -r '.db.dsn' "$CONFIG_FILE")
SCROLL_CHAIN=$(jq -r '.contracts.scroll_chain' "$CONFIG_FILE")
L2_ENDPOINT=$(jq -r '.e2e.l2_rpc' "$CONFIG_FILE")
VALIDIUM_MODE=$(jq -r '.relayer.validium_mode' "$CONFIG_FILE")
MIN_CODEC=$(jq -r '.relayer.min_codec_version' "$CONFIG_FILE")
CHAIN_MONITOR=$(jq -r '.relayer.chain_monitor_enabled' "$CONFIG_FILE")
GENESIS=$(jq -r '.genesis' "$CONFIG_FILE")

# Use hardcoded dev keys for shadow testing
# Anvil default account #0 (commit) and the prover/finalize EOA
COMMIT_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
FINALIZE_KEY="0x01f1e12ee33f91d63172c3d51baa3cecb4469284b0ab45eed48e57fb5329ac4d"

# ─── Render config ───────────────────────────────────────────────────────────
RELAYER_CONFIG="${SCRIPT_DIR}/../.work/relayer-${CONFIG}.json"
mkdir -p "$(dirname "$RELAYER_CONFIG")"

log_info "Rendering relayer config..."

# Simple envsubst-style rendering
export ANVIL_RPC DB_DSN SCROLL_CHAIN L2_ENDPOINT COMMIT_KEY FINALIZE_KEY
export VALIDIUM_MODE CHAIN_MONITOR

sed \
    -e "s|{{ANVIL_RPC}}|$ANVIL_RPC|g" \
    -e "s|{{DB_DSN}}|$DB_DSN|g" \
    -e "s|{{SCROLL_CHAIN}}|$SCROLL_CHAIN|g" \
    -e "s|{{L2_ENDPOINT}}|$L2_ENDPOINT|g" \
    -e "s|{{COMMIT_KEY}}|$COMMIT_KEY|g" \
    -e "s|{{FINALIZE_KEY}}|$FINALIZE_KEY|g" \
    -e "s|{{VALIDIUM_MODE}}|$VALIDIUM_MODE|g" \
    -e "s|{{CHAIN_MONITOR_ENABLED}}|$CHAIN_MONITOR|g" \
    "${SCRIPT_DIR}/../configs/relayer.json.template" > "$RELAYER_CONFIG"

log_ok "  Config written to $RELAYER_CONFIG"

# ─── Build relayer ────────────────────────────────────────────────────────────
# Resolve genesis path relative to repo root
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

RELAYER_BIN="${REPO_ROOT}/rollup/build/bin/rollup_relayer"

if [[ "$BUILD" == "true" || ! -f "$RELAYER_BIN" ]]; then
    log_info "Building relayer..."
    cd "${REPO_ROOT}/rollup"
    go build -o build/bin/rollup_relayer ./cmd/rollup_relayer
    log_ok "  Built: $RELAYER_BIN"
fi
if [[ ! "$GENESIS" = /* ]]; then
    GENESIS="$REPO_ROOT/$GENESIS"
fi

# ─── Launch relayer ───────────────────────────────────────────────────────────
log_info "Starting relayer..."

if [[ "$DOCKER" == "true" ]]; then
    log_warn "Docker mode not yet implemented, falling back to bare metal"
fi

# Kill any existing relayer
pkill -f "rollup_relayer.*relayer-${CONFIG}" 2>/dev/null || true
sleep 1

nohup "$RELAYER_BIN" \
    --config "$RELAYER_CONFIG" \
    --genesis "$GENESIS" \
    --min-codec-version "$MIN_CODEC" \
    --verbosity 3 \
    > "${SCRIPT_DIR}/../.work/relayer-${CONFIG}.log" 2>&1 &

RELAYER_PID=$!
echo "$RELAYER_PID" > "${SCRIPT_DIR}/../.work/relayer-${CONFIG}.pid"

log_ok "  Relayer started (PID $RELAYER_PID)"
log_info "  Logs: ${SCRIPT_DIR}/../.work/relayer-${CONFIG}.log"

# Give it a moment to start
sleep 3

# Quick health check
if ! kill -0 "$RELAYER_PID" 2>/dev/null; then
    log_error "Relayer exited immediately! Check logs."
    tail -20 "${SCRIPT_DIR}/../.work/relayer-${CONFIG}.log"
    exit 1
fi

log_ok "Relayer is running"
