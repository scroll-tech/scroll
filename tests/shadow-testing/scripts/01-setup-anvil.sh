#!/usr/bin/env bash
# Setup Anvil shadow fork for Scroll testing
# Usage: ./01-setup-anvil.sh [options]
#
# Options:
#   --fork-url URL          Ethereum RPC to fork from (default: mainnet)
#   --fork-block NUM        Block number to fork at
#   --anvil-rpc URL         Anvil RPC endpoint (default: http://localhost:18545)
#   --state-file PATH       Save Anvil state to this file after setup
#   --last-finalized NUM    Reset lastFinalizedBatchIndex to this value
#   --last-committed NUM    Reset lastCommittedBatchIndex to this value (default: last-finalized)
#   --committed-batch-hash HASH  Set committedBatches[last-committed] to this hash
#   --next-queue NUM        Reset nextUnfinalizedQueueIndex to this value
#   --deployed-verifier ADDR  Address of ZkEvmVerifierPostFeynman to register
#   --prover-eoa ADDR       EOA to authorize as prover
#   --commit-eoa ADDR       EOA to authorize as sequencer (optional)
#   --owner ADDR            Contract owner address for impersonation
#   --no-anvil              Skip starting Anvil (assume already running)
#   -h, --help              Show this help

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/anvil-utils.sh"

# ─── Defaults ────────────────────────────────────────────────────────────────
FORK_URL="${FORK_URL:-https://eth-mainnet.g.alchemy.com/v2/demo}"
FORK_BLOCK="${FORK_BLOCK:-25202217}"
ANVIL_RPC="${ANVIL_RPC:-http://localhost:18545}"
STATE_FILE=""
LAST_FINALIZED="${LAST_FINALIZED:-517760}"
LAST_COMMITTED="${LAST_COMMITTED:-}"
COMMITTED_BATCH_HASH="${COMMITTED_BATCH_HASH:-}"
NEXT_QUEUE="${NEXT_QUEUE:-0}"

# Mainnet contract addresses (can be overridden for Sepolia)
SCROLL_CHAIN="${SCROLL_CHAIN:-0xa13BAF47339d63B743e7Da8741db5456DAc1E556}"
L1_MSG_QUEUE_V2="${L1_MSG_QUEUE_V2:-0x56971da63A3C0205184FEF096E9ddFc7A8C2D18a}"
ROLLUP_VERIFIER="${ROLLUP_VERIFIER:-0x4CEA3E866e7c57fD75CB0CA3E9F5f1151D4Ead3F}"
DEPLOYED_VERIFIER="${DEPLOYED_VERIFIER:-0xb1F2C5c1ea2885278a1070350d12d3D8824265B0}"
OWNER="${OWNER:-0x798576400F7D662961BA15C6b3F3d813447a26a6}"
PROVER_EOA="${PROVER_EOA:-0x410E7FD80a3Fc1E62A4D3450d11b71b812006eB9}"
COMMIT_EOA="${COMMIT_EOA:-}"
CODEC_VERSION="${CODEC_VERSION:-10}"

NO_ANVIL=false
ANVIL_PID=""

# ─── Parse args ──────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --fork-url)       FORK_URL="$2"; shift 2 ;;
        --fork-block)     FORK_BLOCK="$2"; shift 2 ;;
        --anvil-rpc)      ANVIL_RPC="$2"; shift 2 ;;
        --state-file)     STATE_FILE="$2"; shift 2 ;;
        --last-finalized) LAST_FINALIZED="$2"; shift 2 ;;
        --last-committed) LAST_COMMITTED="$2"; shift 2 ;;
        --committed-batch-hash) COMMITTED_BATCH_HASH="$2"; shift 2 ;;
        --next-queue)     NEXT_QUEUE="$2"; shift 2 ;;
        --deployed-verifier) DEPLOYED_VERIFIER="$2"; shift 2 ;;
        --prover-eoa)     PROVER_EOA="$2"; shift 2 ;;
        --commit-eoa)     COMMIT_EOA="$2"; shift 2 ;;
        --owner)          OWNER="$2"; shift 2 ;;
        --no-anvil)       NO_ANVIL=true; shift ;;
        --scroll-chain)   SCROLL_CHAIN="$2"; shift 2 ;;
        --l1-msg-queue)   L1_MSG_QUEUE_V2="$2"; shift 2 ;;
        --rollup-verifier) ROLLUP_VERIFIER="$2"; shift 2 ;;
        --db-dsn)         DB_DSN="$2"; shift 2 ;;
        --codec-version)  CODEC_VERSION="$2"; shift 2 ;;
        -h|--help)
            sed -n '2,20p' "$0"
            exit 0
            ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

# If last-committed not provided, default to last-finalized (mainnet behavior)
# For Sepolia shadow forks, set last-committed = last-finalized + 1
# NOTE: This must run AFTER argument parsing, because LAST_FINALIZED may be overridden by --last-finalized.
LAST_COMMITTED="${LAST_COMMITTED:-$LAST_FINALIZED}"

# ─── Validate deps ───────────────────────────────────────────────────────────
require_cmd cast
require_cmd anvil

# ─── Step 1: Start Anvil ─────────────────────────────────────────────────────
if [[ "$NO_ANVIL" == "false" ]]; then
    log_info "Starting Anvil fork..."
    log_info "  Fork URL:   $FORK_URL"
    log_info "  Fork Block: $FORK_BLOCK"
    log_info "  RPC:        $ANVIL_RPC"

    # Kill any existing Anvil on the same port
    anvil_port="${ANVIL_RPC##*:}"
    existing_pid=$(lsof -ti :"$anvil_port" 2>/dev/null || true)
    if [[ -n "$existing_pid" ]]; then
        log_warn "Killing existing Anvil on port $anvil_port (PID $existing_pid)"
        kill "$existing_pid" 2>/dev/null || true
        sleep 2
    fi

    setsid nohup anvil \
        --fork-url "$FORK_URL" \
        --fork-block-number "$FORK_BLOCK" \
        --block-time 12 \
        --port "$anvil_port" \
        --host 0.0.0.0 \
        ${STATE_FILE:+--state "$STATE_FILE"} \
        >/dev/null 2>&1 &
    ANVIL_PID=$!

    log_info "Anvil started (PID $ANVIL_PID)"
    sleep 3
else
    log_info "Skipping Anvil startup (using existing instance)"
fi

wait_for_anvil "$ANVIL_RPC"

# ─── Step 2: Reset ScrollChain miscData ──────────────────────────────────────
log_info "Resetting ScrollChain state..."
log_info "  lastFinalizedBatchIndex → $LAST_FINALIZED"

# ScrollChainMiscData is packed into one slot (slot 161):
#   bytes 0-7:   lastCommittedBatchIndex (uint64)
#   bytes 8-15:  lastFinalizedBatchIndex (uint64)
#   bytes 16-19: lastFinalizeTimestamp   (uint32)
#   byte 20:     flags                   (uint8)
#   bytes 21-31: reserved                (uint88)
# We set committed and finalized, zero out timestamp & flags.
committed_hex=$(printf '%016x' "$LAST_COMMITTED")
finalized_hex=$(printf '%016x' "$LAST_FINALIZED")
# ScrollChainMiscData layout (32 bytes), little-endian:
#   bytes 0-7:   lastCommittedBatchIndex (uint64 LE)  - 16 hex
#   bytes 8-15:  lastFinalizedBatchIndex (uint64 LE)  - 16 hex
#   bytes 16-19: lastFinalizeTimestamp   (uint32)     - 8 hex
#   byte 20:     flags                   (uint8)      - 2 hex
#   bytes 21-31: reserved                (uint88)     - 22 hex
# Total: 64 hex chars. We zero out timestamp & flags.
new_miscdata="0x00000000000000000000000000000000${finalized_hex}${committed_hex}"

set_storage "$SCROLL_CHAIN" "0x00000000000000000000000000000000000000000000000000000000000000a1" "$new_miscdata" "$ANVIL_RPC"

# If a committed batch hash is provided, set committedBatches[lastCommittedBatchIndex]
# This is required for shadow forks where we need the parent batch hash
# to match when the relayer calls commitBatches.
# If --db-dsn is provided but no hash, auto-fetch from DB.
if [[ -z "$COMMITTED_BATCH_HASH" || "$COMMITTED_BATCH_HASH" == "0x0000000000000000000000000000000000000000000000000000000000000000" ]]; then
    if [[ -n "${DB_DSN:-}" ]]; then
        log_info "  Fetching committedBatches[$LAST_COMMITTED] hash from DB..."
        COMMITTED_BATCH_HASH=$(psql "$DB_DSN" -Atq -c "
            SELECT hash FROM batch WHERE index = $LAST_COMMITTED
        " 2>/dev/null | tr -d ' ')
        if [[ -n "$COMMITTED_BATCH_HASH" && "$COMMITTED_BATCH_HASH" != "NULL" ]]; then
            log_info "  Found hash: $COMMITTED_BATCH_HASH"
        else
            log_warn "  Batch $LAST_COMMITTED not found in DB; committedBatches will not be seeded"
            COMMITTED_BATCH_HASH=""
        fi
    fi
fi

if [[ -n "$COMMITTED_BATCH_HASH" && "$COMMITTED_BATCH_HASH" != "0x0000000000000000000000000000000000000000000000000000000000000000" ]]; then
    log_info "  Setting committedBatches[$LAST_COMMITTED] = $COMMITTED_BATCH_HASH"
    # committedBatches is mapping(uint256 => bytes32) at slot 157
    committed_slot=$(cast index uint256 "$LAST_COMMITTED" 157 2>/dev/null)
    set_storage "$SCROLL_CHAIN" "$committed_slot" "$COMMITTED_BATCH_HASH" "$ANVIL_RPC"
    log_ok "  committedBatches[$LAST_COMMITTED] set"
fi

# Verify
actual_finalized=$(cast call "$SCROLL_CHAIN" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$ANVIL_RPC" 2>/dev/null)
actual_committed=$(cast call "$SCROLL_CHAIN" "miscData()(uint64,uint64,uint32,uint8,uint88)" --rpc-url "$ANVIL_RPC" 2>/dev/null | cut -d',' -f1 | tr -d ' ')
log_ok "  lastFinalizedBatchIndex = $actual_finalized"
log_ok "  lastCommittedBatchIndex = $actual_committed"

# ─── Step 3: Reset L1MessageQueueV2 ──────────────────────────────────────────
log_info "Resetting L1MessageQueueV2..."
log_info "  nextUnfinalizedQueueIndex → $NEXT_QUEUE"

# Slot 104 holds nextUnfinalizedQueueIndex (uint256)
queue_hex=$(encode_uint256 "$NEXT_QUEUE")
set_storage "$L1_MSG_QUEUE_V2" "0x0000000000000000000000000000000000000000000000000000000000000068" "$queue_hex" "$ANVIL_RPC"

actual_queue=$(cast call "$L1_MSG_QUEUE_V2" "nextUnfinalizedQueueIndex()(uint256)" --rpc-url "$ANVIL_RPC" 2>/dev/null)
log_ok "  nextUnfinalizedQueueIndex = $actual_queue"

# ─── Step 4: Deploy / copy verifier ──────────────────────────────────────────
if [[ -n "$DEPLOYED_VERIFIER" && "$DEPLOYED_VERIFIER" != "0x0000000000000000000000000000000000000000" ]]; then
    log_info "Using provided verifier..."
    log_info "  Verifier: $DEPLOYED_VERIFIER"
else
    log_info "No deployed verifier provided. Attempting to copy from known shadow-compatible verifier..."
    # Copy mainnet shadow verifier (0xb1F2...) to a deterministic address on this Anvil fork
    SHADOW_VERIFIER="0xb1F2C5c1ea2885278a1070350d12d3D8824265B0"
    SHADOW_PLONK="0x4A2CA4AB67922F9a9212C6ab20eFF23bdE132263"
    
    # These addresses must exist on the source RPC (mainnet Anvil from previous test)
    SRC_RPC="${SRC_RPC:-http://localhost:18545}"
    
    verifier_code=$(cast code "$SHADOW_VERIFIER" --rpc-url "$SRC_RPC" 2>/dev/null || echo "")
    plonk_code=$(cast code "$SHADOW_PLONK" --rpc-url "$SRC_RPC" 2>/dev/null || echo "")
    
    if [[ -n "$verifier_code" && -n "$plonk_code" ]]; then
        cast rpc anvil_setCode "$SHADOW_PLONK" "$plonk_code" --rpc-url "$ANVIL_RPC" >/dev/null 2>&1
        cast rpc anvil_setCode "$SHADOW_VERIFIER" "$verifier_code" --rpc-url "$ANVIL_RPC" >/dev/null 2>&1
        DEPLOYED_VERIFIER="$SHADOW_VERIFIER"
        log_ok "  Copied verifier to $DEPLOYED_VERIFIER"
    else
        log_warn "  Could not copy verifier from $SRC_RPC"
        log_warn "  You will need to manually deploy a verifier matching your proofs"
    fi
fi

# ─── Step 4b: Set owner balance (needed for impersonated transactions) ──────
log_info "Setting owner balance..."
set_balance "$OWNER" "0x56bc75e2d63100000" "$ANVIL_RPC"
log_ok "  Owner balance = 100 ETH"

# ─── Step 4c: Clear EIP-7702 delegation from commit EOA ─────────────────────
if [[ -n "$COMMIT_EOA" ]]; then
    commit_code=$(cast code "$COMMIT_EOA" --rpc-url "$ANVIL_RPC" 2>/dev/null)
    if [[ "$commit_code" == 0xef01* ]]; then
        log_warn "  Commit EOA has EIP-7702 delegation, clearing..."
        cast rpc anvil_setCode "$COMMIT_EOA" "0x" --rpc-url "$ANVIL_RPC" >/dev/null 2>&1
        log_ok "  EIP-7702 delegation cleared"
    fi
fi

# ─── Step 5: Register verifier ───────────────────────────────────────────────
if [[ -n "$DEPLOYED_VERIFIER" && "$DEPLOYED_VERIFIER" != "0x0000000000000000000000000000000000000000" ]]; then
    log_info "Registering verifier..."
    log_info "  Verifier: $DEPLOYED_VERIFIER"
    log_info "  Codec:    $CODEC_VERSION"

    impersonate "$OWNER" "$ANVIL_RPC"

    # startBatchIndex must be > lastFinalizedBatchIndex to pass contract checks
    start_batch_index=$((LAST_FINALIZED + 1))

    # Use eth_sendTransaction directly to avoid cast send --unlocked bugs with impersonation
    verifier_calldata=$(cast calldata "updateVerifier(uint256,uint64,address)" "$CODEC_VERSION" "$start_batch_index" "$DEPLOYED_VERIFIER")
    cast rpc eth_sendTransaction \
        "{\"from\":\"$OWNER\",\"to\":\"$ROLLUP_VERIFIER\",\"data\":\"$verifier_calldata\",\"gas\":\"0x4c4b40\"}" \
        --rpc-url "$ANVIL_RPC" >/dev/null 2>&1

    stop_impersonate "$OWNER" "$ANVIL_RPC"

    # latestVerifier returns (uint64 startBatchIndex, address verifier)
    registered=$(cast call "$ROLLUP_VERIFIER" "latestVerifier(uint256)" "$CODEC_VERSION" --rpc-url "$ANVIL_RPC" 2>/dev/null | sed 's/0x//' | cut -c65-128 | sed 's/^0*//')
    log_ok "  latestVerifier[$CODEC_VERSION] = 0x$registered (startBatchIndex=$start_batch_index)"
else
    log_warn "Skipping verifier registration (no verifier address available)"
fi

# ─── Step 5: Authorize prover ────────────────────────────────────────────────
log_info "Authorizing prover EOA..."
log_info "  Prover: $PROVER_EOA"

impersonate "$OWNER" "$ANVIL_RPC"

cast send "$SCROLL_CHAIN" \
    "addProver(address)" "$PROVER_EOA" \
    --from "$OWNER" --rpc-url "$ANVIL_RPC" --unlocked >/dev/null 2>&1

stop_impersonate "$OWNER" "$ANVIL_RPC"

is_prover=$(cast call "$SCROLL_CHAIN" "isProver(address)(bool)" "$PROVER_EOA" --rpc-url "$ANVIL_RPC" 2>/dev/null)
log_ok "  isProver[$PROVER_EOA] = $is_prover"

# ─── Step 6: Authorize commit EOA as sequencer (optional) ────────────────────
if [[ -n "$COMMIT_EOA" ]]; then
    log_info "Authorizing commit EOA as sequencer..."
    log_info "  Sequencer: $COMMIT_EOA"

    impersonate "$OWNER" "$ANVIL_RPC"

    if cast send --gas-limit 5000000 "$SCROLL_CHAIN" \
        "addSequencer(address)" "$COMMIT_EOA" \
        --from "$OWNER" --rpc-url "$ANVIL_RPC" --unlocked >/dev/null 2>&1; then
        stop_impersonate "$OWNER" "$ANVIL_RPC"
        is_seq=$(cast call "$SCROLL_CHAIN" "isSequencer(address)(bool)" "$COMMIT_EOA" --rpc-url "$ANVIL_RPC" 2>/dev/null)
        log_ok "  isSequencer[$COMMIT_EOA] = $is_seq"
    else
        stop_impersonate "$OWNER" "$ANVIL_RPC"
        log_warn "  addSequencer failed (account may have code, e.g. EIP-7702). Skipping."
    fi
fi

# ─── Step 7: Set balances ────────────────────────────────────────────────────
log_info "Setting balances..."
set_balance "$PROVER_EOA" "0x56bc75e2d63100000" "$ANVIL_RPC"
log_ok "  Prover balance = 100 ETH"

if [[ -n "$COMMIT_EOA" ]]; then
    set_balance "$COMMIT_EOA" "0x56bc75e2d63100000" "$ANVIL_RPC"
    log_ok "  Commit balance = 100 ETH"
fi

# ─── Step 8: Save state ──────────────────────────────────────────────────────
if [[ -n "$STATE_FILE" ]]; then
    log_info "Saving Anvil state to $STATE_FILE..."
    cast rpc anvil_dumpState --rpc-url "$ANVIL_RPC" > "$STATE_FILE"
    log_ok "  State saved ($(wc -c < "$STATE_FILE" | numfmt --to=iec-i))"
fi

log_ok "Anvil setup complete!"

# If we started Anvil, keep it running in foreground
if [[ "$NO_ANVIL" == "false" && -n "$ANVIL_PID" ]]; then
    log_info "Anvil running in background (PID $ANVIL_PID)"
    echo "$ANVIL_PID" > "${SCRIPT_DIR}/../../.work/anvil.pid"
fi
