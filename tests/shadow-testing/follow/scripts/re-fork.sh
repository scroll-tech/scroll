#!/usr/bin/env bash
# re-fork.sh — recovery procedure for restarting/re-forking Anvil mid-run.
#
# Keeps coordinator, provers and the sync/sweeper/monitor daemons running.
# Steps (Trap numbers refer to tests/shadow-testing/docs/TROUBLESHOOTING.md):
#   1. stop the relayer (avoid error spam while Anvil is down)
#   2. kill Anvil, re-fork at the CURRENT latest L1 block (01-setup-anvil.sh)
#   3. redeploy the verifier wrapper (03-deploy-verifier.sh) — anvil_setCode
#      copies preserve stale immutables, so a full redeploy is mandatory
#      (AGENTS.md: "anvil_setCode Does NOT Reset Immutables")
#   4. re-fund EOAs — Trap 10: Anvil restart resets balances (handled by
#      01-setup-anvil.sh step 7, which sets owner/prover/commit balances)
#   5. re-mirror L1 queue rolling hashes — Trap 23: messageRollingHashes +
#      nextCrossDomainMessageIndex must be re-mirrored onto the fresh fork
#   6. restart the relayer (06-run-relayer.sh)
#
# Caveat: bundles already finalized on the previous fork keep rollup_status=5
# in the shadow DB and are NOT re-finalized; lastFinalizedBatchIndex is set
# from mainnet's value at the new fork block. Step 5b clears pending_transaction
# (Trap 27 nonce desync) and realigns batch/bundle rollup_status to the fresh
# fork's committed/finalized boundaries.
#
# Usage: ./re-fork.sh [--config configs/mainnet.json]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
LIB_DIR="$(cd "${SCRIPT_DIR}/../../lib" && pwd)"
source "${LIB_DIR}/anvil-utils.sh"

CONFIG_FILE="${CONFIG_FILE:-${PROJECT_ROOT}/configs/mainnet.json}"
# .work is shared by both modes and stays at tests/shadow-testing/.work.
WORK_DIR="${WORK_DIR:-${PROJECT_ROOT}/../.work}"
MIN_CODEC="${MIN_CODEC:-10}"
NEXT_QUEUE="${NEXT_QUEUE:-0}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --config) CONFIG_FILE="$2"; shift 2 ;;
        -h|--help) sed -n '2,21p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

[[ -f "$CONFIG_FILE" ]] || { log_error "Config not found: $CONFIG_FILE"; exit 1; }
CONFIG_FILE="$(cd "$(dirname "$CONFIG_FILE")" && pwd)/$(basename "$CONFIG_FILE")"

FORK_URL=$(jq -r '.fork.url' "$CONFIG_FILE")
ANVIL_RPC=$(jq -r '.fork.anvil_rpc' "$CONFIG_FILE")
SHADOW_DSN=$(jq -r '.db.dsn' "$CONFIG_FILE")
SCROLL_CHAIN=$(jq -r '.contracts.scroll_chain' "$CONFIG_FILE")
L1_MSG_QUEUE=$(jq -r '.contracts.l1_message_queue_v2' "$CONFIG_FILE")
ROLLUP_VERIFIER=$(jq -r '.contracts.rollup_verifier' "$CONFIG_FILE")
OWNER=$(jq -r '.contracts.owner' "$CONFIG_FILE")
PROVER_EOA=$(jq -r '.accounts.prover_eoa' "$CONFIG_FILE")
COMMIT_EOA=$(jq -r '.accounts.commit_eoa' "$CONFIG_FILE")
CONFIG_NAME=$(basename "$CONFIG_FILE" .json)

# ─── 1. Stop relayer (avoid revert spam while Anvil is down) ─────────────────
log_info "=== 1. Stopping relayer ==="
pkill -f "rollup_relayer.*relayer-${CONFIG_NAME}" 2>/dev/null || true
rm -f "${WORK_DIR}/relayer-${CONFIG_NAME}.pid"
sleep 1

# ─── 2. Kill Anvil and re-fork at the current latest block ───────────────────
log_info "=== 2. Re-forking Anvil at latest block ==="
if [[ -f "${WORK_DIR}/anvil.pid" ]] && kill -0 "$(cat "${WORK_DIR}/anvil.pid")" 2>/dev/null; then
    kill "$(cat "${WORK_DIR}/anvil.pid")" 2>/dev/null || true
else
    lsof -ti :18545 2>/dev/null | xargs -r kill 2>/dev/null || true
fi
rm -f "${WORK_DIR}/anvil.pid"
sleep 2

FORK_BLOCK_NUMBER=$(cast block-number --rpc-url "$FORK_URL")
LAST_FINALIZED=$(cast call "$SCROLL_CHAIN" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$FORK_URL" | awk '{print $1}')
log_info "  fork block:    $FORK_BLOCK_NUMBER"
log_info "  lastFinalized: $LAST_FINALIZED (mainnet value at fork block)"

STATE_FILE="${WORK_DIR}/anvil-${CONFIG_NAME}.state.json"
if [[ -f "$STATE_FILE" ]]; then
    # A stale state file would make Anvil resurrect the old fork; back it up.
    STATE_BAK="${STATE_FILE}.bak.$(date +%Y%m%d%H%M%S)"
    log_warn "  backing up existing state file -> $STATE_BAK"
    mv "$STATE_FILE" "$STATE_BAK"
fi

"${LIB_DIR}/01-setup-anvil.sh" \
    --fork-url "$FORK_URL" \
    --fork-block "$FORK_BLOCK_NUMBER" \
    --anvil-rpc "$ANVIL_RPC" \
    --state-file "$STATE_FILE" \
    --last-finalized "$LAST_FINALIZED" \
    --next-queue "$NEXT_QUEUE" \
    --deployed-verifier "" \
    --prover-eoa "$PROVER_EOA" \
    --commit-eoa "$COMMIT_EOA" \
    --owner "$OWNER" \
    --scroll-chain "$SCROLL_CHAIN" \
    --l1-msg-queue "$L1_MSG_QUEUE" \
    --rollup-verifier "$ROLLUP_VERIFIER" \
    --db-dsn "$SHADOW_DSN"

# ─── 3. Redeploy verifier wrapper on the fresh fork ──────────────────────────
# A redeploy is mandatory: copying code with anvil_setCode preserves the old
# wrapper's immutables (verifierDigest1/2, protocolVersion) which are bound to
# a different plonk VK and would fail every verification (0x439cc0cd).
log_info "=== 3. Redeploying verifier wrapper ==="
"${LIB_DIR}/03-deploy-verifier.sh" --config "$CONFIG_FILE"

# ─── 4. Re-fund EOAs (Trap 10) ───────────────────────────────────────────────
# Trap 10: Anvil restart resets account balances. 01-setup-anvil.sh step 7
# already re-funds owner/prover/commit EOAs on the fresh fork — nothing extra
# needed here, but verify the relayer's prover EOA is funded.
log_info "=== 4. Verifying EOA balances (Trap 10) ==="
PROVER_BAL=$(cast balance "$PROVER_EOA" --rpc-url "$ANVIL_RPC")
COMMIT_BAL=$(cast balance "$COMMIT_EOA" --rpc-url "$ANVIL_RPC")
log_info "  prover EOA balance: $PROVER_BAL"
log_info "  commit EOA balance: $COMMIT_BAL"
[[ "$PROVER_BAL" != "0" && "$COMMIT_BAL" != "0" ]] || { log_error "EOA funding missing after re-fork!"; exit 1; }

# ─── 5. Re-mirror L1 queue rolling hashes (Trap 23) ──────────────────────────
# Trap 23: bundles popping L1 messages enqueued after the fork block fail with
# VerificationFailed (0x439cc0cd) / ErrorFinalizedIndexTooLarge (0x16465978)
# unless messageRollingHashes and nextCrossDomainMessageIndex are re-mirrored.
log_info "=== 5. Re-mirroring L1 queue rolling hashes (Trap 23) ==="
FORK_RPC="$ANVIL_RPC" python3 "${LIB_DIR}/sync-queue-hashes.py"

# ─── 5b. Reset relayer nonce state + rollup statuses for the fresh fork ──────
# Trap 27: the fresh fork resets account nonces to forked mainnet values (0 for
# the local sender keys), but pending_transaction keeps the old high nonces —
# the relayer would sign with a nonce gap and every tx would queue forever.
log_info "=== 5b. Clearing pending_transaction + realigning rollup_status ==="
MISC_RAW=$(cast call "$SCROLL_CHAIN" 0x06582acb --rpc-url "$ANVIL_RPC")
FORK_COMMITTED=$((16#${MISC_RAW:2:64}))
log_info "  fresh fork lastCommittedBatchIndex: $FORK_COMMITTED"
psql "$SHADOW_DSN" -Atq --set ON_ERROR_STOP=1 -c "
    DELETE FROM pending_transaction;
    UPDATE batch  SET rollup_status = 5, proving_status = 4
      WHERE index <= ${LAST_FINALIZED} AND (rollup_status <> 5 OR proving_status <> 4);
    UPDATE bundle SET rollup_status = 5, proving_status = 4
      WHERE end_batch_index <= ${LAST_FINALIZED} AND (rollup_status <> 5 OR proving_status <> 4);
    UPDATE batch  SET rollup_status = 3, commit_tx_hash = NULL, committed_at = NULL,
                      finalize_tx_hash = NULL, finalized_at = NULL
      WHERE index > ${LAST_FINALIZED} AND index <= ${FORK_COMMITTED} AND rollup_status <> 3;
    UPDATE batch  SET rollup_status = 1, commit_tx_hash = NULL, committed_at = NULL,
                      finalize_tx_hash = NULL, finalized_at = NULL
      WHERE index > ${FORK_COMMITTED} AND rollup_status <> 1;
    UPDATE bundle SET rollup_status = 1, finalize_tx_hash = NULL, finalized_at = NULL
      WHERE end_batch_index > ${LAST_FINALIZED} AND rollup_status <> 1;
"
log_ok "  nonce state cleared; rollup statuses aligned to fresh fork"

# ─── 6. Restart relayer ──────────────────────────────────────────────────────
log_info "=== 6. Restarting relayer ==="
MIN_CODEC="$MIN_CODEC" "${LIB_DIR}/06-run-relayer.sh" --config "$CONFIG_NAME"

log_ok "Re-fork complete. Coordinator, provers and sync daemons were left running."
log_info "Note: update .work/follow-run.env FORK_BLOCK if you track it:"
log_info "  sed -i 's/^FORK_BLOCK=.*/FORK_BLOCK=${FORK_BLOCK_NUMBER}/' .work/follow-run.env"
