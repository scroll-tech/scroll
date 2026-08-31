#!/usr/bin/env bash
# 31-canary-rollback.sh — roll back a canary parallel upgrade ("as if the
# upgrade never happened").
#
# Undoes 30-canary-upgrade.sh at the SAME boundary batch N (read from
# .work/follow-run.env CANARY_* keys):
#
#   a. sanity checks (follow-run.env carries CANARY_AT_BATCH/CANARY_OLD_WRAPPER,
#      Anvil/DB reachable)
#   b. stop the new coordinator_api/coordinator_cron + provers
#      (Anvil, relayer, poll-sync, sweeper, monitor KEEP RUNNING)
#   c. impersonate the MVRV owner and call updateVerifier(10, N, OLD_WRAPPER)
#      — same startBatchIndex as the upgrade, so the latest verifier is
#      swapped in place (no new legacyVerifiers entry)
#   d. assert getVerifier(10, N) == OLD_WRAPPER
#   e. apply quarantined remote proofs to unfinalized bundles with
#      end_batch_index >= N (overwrites any local NEW-stack proofs, which the
#      restored old wrapper could never verify); >= N bundles with NO
#      quarantined proof yet are reset to unproven so the relayer never sends
#      a new-stack proof to the old wrapper
#   f. clear the proof-import boundary (.work/canary.env) back to +inf, so
#      poll-sync applies every quarantined proof as it arrives and
#      finalization of NEW bundles >= N continues unattended
#
# Afterwards 30-canary-upgrade.sh can be re-run to upgrade again (it
# recomputes N; the in-place legacy entry at the old N does not interfere).
#
# Usage: ./31-canary-rollback.sh [--config configs/mainnet.json] [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
LIB_DIR="$(cd "${SCRIPT_DIR}/../../lib" && pwd)"
source "${LIB_DIR}/anvil-utils.sh"

# ─── Defaults (env-overridable) ──────────────────────────────────────────────
CONFIG_FILE="${CONFIG_FILE:-${PROJECT_ROOT}/configs/mainnet.json}"
WORK_DIR="${WORK_DIR:-${PROJECT_ROOT}/../.work}"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)  CONFIG_FILE="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        -h|--help) sed -n '2,28p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

[[ -f "$CONFIG_FILE" ]] || { log_error "Config not found: $CONFIG_FILE"; exit 1; }
CONFIG_FILE="$(cd "$(dirname "$CONFIG_FILE")" && pwd)/$(basename "$CONFIG_FILE")"

pid_alive() {
    local pidfile="$1" pid
    [[ -f "$pidfile" ]] || return 1
    pid=$(cat "$pidfile" 2>/dev/null || true)
    [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

stop_component() {  # label pidfile cmdline-substring (same semantics as 20-upgrade.sh)
    local label="$1" pidfile="$2" expect="$3" pid
    if [[ -f "$pidfile" ]]; then
        pid=$(cat "$pidfile" 2>/dev/null || true)
        if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
            if $DRY_RUN; then log_info "[dry-run] stop $label (pid $pid)"; return 0; fi
            kill -- "-$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
            for _ in $(seq 1 10); do kill -0 "$pid" 2>/dev/null || break; sleep 1; done
            kill -9 -- "-$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
            log_ok "  stopped $label (pid $pid)"
            $DRY_RUN || rm -f "$pidfile"
            # Do NOT return here: `setsid nohup CMD &` can record a setsid
            # intermediate that becomes a zombie while the real process lives
            # on under a different pid — fall through to the pattern sweep.
        fi
        $DRY_RUN || rm -f "$pidfile"
    fi
    local pids
    pids=$(pgrep -f "$expect" 2>/dev/null || true)
    if [[ -n "$pids" ]]; then
        if $DRY_RUN; then log_info "[dry-run] pkill -f '$expect'"; return 0; fi
        echo "$pids" | xargs -r kill 2>/dev/null || true
        sleep 2
        pgrep -f "$expect" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
        log_ok "  stopped $label (pattern '$expect')"
    fi
}

# ─── Load config + canary state ──────────────────────────────────────────────
ANVIL_RPC=$(jq -r '.fork.anvil_rpc' "$CONFIG_FILE")
SHADOW_DSN=$(jq -r '.db.dsn' "$CONFIG_FILE")
MVRV=$(jq -r '.contracts.rollup_verifier' "$CONFIG_FILE")
OWNER=$(jq -r '.contracts.owner' "$CONFIG_FILE")

FOLLOW_ENV="${WORK_DIR}/follow-run.env"
N=$(grep '^CANARY_AT_BATCH=' "$FOLLOW_ENV" 2>/dev/null | cut -d= -f2 || true)
OLD_WRAPPER=$(grep '^CANARY_OLD_WRAPPER=' "$FOLLOW_ENV" 2>/dev/null | cut -d= -f2 || true)
if [[ -z "$N" || -z "$OLD_WRAPPER" ]]; then
    log_error "CANARY_AT_BATCH / CANARY_OLD_WRAPPER missing in $FOLLOW_ENV —"
    log_error "run 30-canary-upgrade.sh first (nothing to roll back)."
    exit 1
fi

# ─── a. Sanity checks ────────────────────────────────────────────────────────
log_info "=== a. Sanity checks ==="
require_cmd jq; require_cmd cast; require_cmd psql

cast block-number --rpc-url "$ANVIL_RPC" >/dev/null 2>&1 \
    || { log_error "Anvil RPC unreachable: $ANVIL_RPC"; exit 1; }
log_ok "  Anvil reachable"
psql "$SHADOW_DSN" -Atq -c "SELECT 1" >/dev/null || { log_error "shadow DB unreachable"; exit 1; }
log_ok "  shadow DB reachable"
log_info "  rolling back canary upgrade at batch N=$N to old wrapper $OLD_WRAPPER"

# ─── b. Stop the new coordinator + provers (daemons keep running) ────────────
log_info "=== b. Stop new coordinator + provers ==="
for pf in "${WORK_DIR}"/prover-*/prover.pid; do
    [[ -f "$pf" ]] || continue
    gpu=$(basename "$(dirname "$pf")")
    stop_component "$gpu" "$pf" "prover.json"
done
if ! $DRY_RUN; then pkill -f "target/release/prover" 2>/dev/null || true; fi
# Docker-mode provers: pidfile kills above normally suffice (host PIDs), but
# remove any lingering containers so the next start cannot hit a name clash.
if ! $DRY_RUN && command -v docker >/dev/null 2>&1 \
    && docker ps -aq --filter 'name=^shadow-prover-' 2>/dev/null | grep -q .; then
    docker rm -f $(docker ps -aq --filter 'name=^shadow-prover-') >/dev/null 2>&1 || true
    log_info "  removed shadow-prover-* containers"
fi
stop_component "coordinator_cron" "${WORK_DIR}/coordinator-cron.pid" "coordinator_cron"
stop_component "coordinator_api"  "${WORK_DIR}/coordinator-api.pid"  "coordinator_api"

# ─── c. Restore the OLD wrapper at the same startBatch (genuine path) ────────
log_info "=== c. updateVerifier(10, $N, OLD_WRAPPER) ==="
if $DRY_RUN; then
    log_info "[dry-run] would call updateVerifier(10, $N, $OLD_WRAPPER) as owner $OWNER"
else
    cast rpc anvil_impersonateAccount "$OWNER" --rpc-url "$ANVIL_RPC" >/dev/null 2>&1 || true
    UPDATE_CALLDATA=$(cast calldata "updateVerifier(uint256,uint64,address)" 10 "$N" "$OLD_WRAPPER")
    UPDATE_TX=$(cast rpc eth_sendTransaction \
        "{\"from\":\"$OWNER\",\"to\":\"$MVRV\",\"data\":\"$UPDATE_CALLDATA\",\"gas\":\"0x4c4b40\"}" \
        --rpc-url "$ANVIL_RPC" 2>/dev/null | tr -d '"')
    if [[ -z "$UPDATE_TX" || "$UPDATE_TX" != 0x* ]]; then
        log_error "updateVerifier eth_sendTransaction failed (impersonated owner $OWNER)"
        exit 1
    fi
    for _ in $(seq 1 20); do
        cast receipt "$UPDATE_TX" --rpc-url "$ANVIL_RPC" >/dev/null 2>&1 && break
        sleep 0.5
    done
    TX_STATUS=$(cast receipt "$UPDATE_TX" status --rpc-url "$ANVIL_RPC" 2>/dev/null || echo "")
    # cast >= 1.6 prints "1 (success)", cast >= 1.7 prints "true"/"false" (Trap 36).
    if ! grep -qiE '^(0x1|1( \(success\))?|true)$' <<<"$TX_STATUS"; then
        log_error "updateVerifier reverted (tx $UPDATE_TX, status '$TX_STATUS')"
        exit 1
    fi
    log_ok "  updateVerifier tx: $UPDATE_TX"
fi

# ─── d. Assert routing restored ──────────────────────────────────────────────
log_info "=== d. Verify MVRV routing ==="
if $DRY_RUN; then
    log_info "[dry-run] would check getVerifier(10, $N) == $OLD_WRAPPER"
else
    ROUTE=$(cast call "$MVRV" "getVerifier(uint256,uint256)(address)" 10 "$N" --rpc-url "$ANVIL_RPC" 2>/dev/null | tr -d ' ' || true)
    log_info "  getVerifier(10, $N) = $ROUTE"
    [[ "${ROUTE,,}" == "${OLD_WRAPPER,,}" ]] || { log_error "batch $N does NOT route back to the old wrapper"; exit 1; }
    log_ok "  routing restored: batches >= $N -> old wrapper $OLD_WRAPPER"
fi

# ─── e. Apply quarantined proofs >= N ────────────────────────────────────────
log_info "=== e. Apply quarantined remote proofs for batches >= $N ==="
APPLY_SQL="
-- Overwrite any local NEW-stack proofs: the restored old wrapper cannot
-- verify them. rollup_status <> 5 keeps already-finalized rows untouched.
UPDATE bundle b SET proof = r.proof, proving_status = 4, proved_at = r.proved_at
  FROM remote_bundle_proof r
  WHERE b.hash = r.bundle_hash
    AND b.rollup_status <> 5
    AND b.end_batch_index >= ${N};
-- Bundles >= N the remote has not proved yet (shadow ran ahead of mainnet):
-- reset to unproven so the relayer never sends a new-stack proof to the old
-- wrapper. 30-canary-upgrade.sh can re-prove them on the next upgrade.
UPDATE bundle SET proof = NULL, proving_status = 1, batch_proofs_status = 1, total_attempts = 0, active_attempts = 0
  WHERE rollup_status <> 5
    AND end_batch_index >= ${N}
    AND proving_status = 4
    AND hash NOT IN (SELECT bundle_hash FROM remote_bundle_proof);
DELETE FROM prover_task WHERE proving_status IN (1,3);
"
if $DRY_RUN; then
    log_info "[dry-run] psql apply SQL:"; echo "$APPLY_SQL"
else
    psql "$SHADOW_DSN" -Atq --set ON_ERROR_STOP=1 -c "$APPLY_SQL" >/dev/null
    log_ok "  quarantined proofs >= $N applied; unbacked new proofs reset"
fi

# ─── f. Clear the proof-import boundary (back to +inf) ───────────────────────
# The poll-sync apply rule only fills bundles with end_batch < boundary. If we
# left the boundary at N after rolling back, mainnet proofs for NEW bundles
# >= N would stay quarantined forever and finalization would stall once the
# step-e backlog drains. Removing the boundary restores Phase-A semantics:
# every quarantined proof is applied as it arrives — "as if the upgrade never
# happened". 30-canary-upgrade.sh writes a fresh boundary on the next upgrade.
log_info "=== f. Clear proof-import boundary ==="
CANARY_ENV_FILE="${WORK_DIR}/canary.env"
if $DRY_RUN; then
    log_info "[dry-run] would remove PROOF_IMPORT_MAX_END_BATCH from $CANARY_ENV_FILE"
else
    if [[ -f "$CANARY_ENV_FILE" ]]; then
        tmp=$(mktemp)
        grep -v '^PROOF_IMPORT_MAX_END_BATCH=' "$CANARY_ENV_FILE" > "$tmp" || true
        mv "$tmp" "$CANARY_ENV_FILE"
    fi
    log_ok "  boundary cleared (poll-sync now applies every quarantined proof)"
fi

echo ""
log_ok "Canary rollback complete — as if the upgrade never happened."
echo ""
echo "State now:"
echo "  - MVRV routes batches >= $N to the OLD wrapper $OLD_WRAPPER again"
echo "  - unfinalized bundles >= $N carry imported production proofs (or are"
echo "    reset to unproven until the remote proof arrives)"
echo "  - relayer/poll-sync/anvil were never stopped — finalization continues"
echo "  - re-run scripts/30-canary-upgrade.sh to upgrade again"
