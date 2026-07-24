#!/usr/bin/env bash
# 20-upgrade.sh — mid-run zk-stack upgrade for follow-mode shadow testing.
#
# Simulates a production zk upgrade WITHOUT resetting the fork or the DB:
# finalization stays continuous across the boundary.
#
#   a. sanity checks (stack running, next config, prover binary)
#   b. compute the boundary batch N:
#        - bundles already proven (old circuit) but not yet finalized keep
#          their proofs and stay on the OLD verifier  (mixed mode)
#        - everything at/after N is reset and re-proven with the NEW stack
#        - if proven and unproven bundles interleave (no clean cut), fall back
#          to a hard cutover at lastFinalized+1 with a warning
#   c. record OLD wrapper = getVerifier(10, N-1) on the forked MVRV
#   d. stop provers + coordinator_api + coordinator_cron
#      (Anvil, relayer, poll-sync, sweeper, monitor KEEP RUNNING)
#   e. reset in-flight/stale task rows >= N; wipe prover-local proof caches
#   f. download NEW verifier assets (verifier.bin / root_verifier_vk /
#      openVmVk.json) and point the coordinator config at them
#   g. restart coordinator_api + coordinator_cron
#   h. deploy new plonk verifier + ZkEvmVerifierPostFeynman wrapper and register
#      it via the GENUINE updateVerifier path (03-deploy-verifier.sh
#      --genuine-register), so the old wrapper moves to legacyVerifiers
#   i. verify MVRV routing: batch N-1 -> old wrapper, batch N -> new wrapper
#   j. restart provers with the new circuit version / S3 base URL
#   k. record the boundary in .work/follow-run.env (report splits on it)
#
# Prerequisites: the operator has already checked out / built the NEW prover
# (and coordinator, if it changed) binaries. Phase 1 should have been brought
# up with `10-follow-up.sh --skip-verifier` (old stack = production release).
#
# Usage: ./20-upgrade.sh --next-config configs/mainnet-next.json
#                        [--config configs/mainnet.json] [--start-batch N] [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
LIB_DIR="$(cd "${SCRIPT_DIR}/../../lib" && pwd)"
source "${LIB_DIR}/anvil-utils.sh"

# ─── Defaults (env-overridable) ──────────────────────────────────────────────
CONFIG_FILE="${CONFIG_FILE:-${PROJECT_ROOT}/configs/mainnet.json}"
NEXT_CONFIG_FILE=""
WORK_DIR="${WORK_DIR:-${PROJECT_ROOT}/../.work}"
COORD_DIR="${COORD_DIR:-${REPO_ROOT}/coordinator/build/bin}"
GPUS="${GPUS:-0,1}"
FORK_NAME="${FORK_NAME:-galileoV2}"
EXPECTED_PROTOCOL_VERSION="${EXPECTED_PROTOCOL_VERSION:-10}"
START_BATCH_OVERRIDE=""
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)      CONFIG_FILE="$2"; shift 2 ;;
        --next-config) NEXT_CONFIG_FILE="$2"; shift 2 ;;
        --start-batch) START_BATCH_OVERRIDE="$2"; shift 2 ;;
        --dry-run)     DRY_RUN=true; shift ;;
        -h|--help)     sed -n '2,33p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

if [[ -z "$NEXT_CONFIG_FILE" ]]; then
    log_error "--next-config is required (e.g. configs/mainnet-next.json)"
    exit 1
fi
[[ -f "$CONFIG_FILE" ]]      || { log_error "Config not found: $CONFIG_FILE"; exit 1; }
[[ -f "$NEXT_CONFIG_FILE" ]] || { log_error "Next config not found: $NEXT_CONFIG_FILE"; exit 1; }
CONFIG_FILE="$(cd "$(dirname "$CONFIG_FILE")" && pwd)/$(basename "$CONFIG_FILE")"
NEXT_CONFIG_FILE="$(cd "$(dirname "$NEXT_CONFIG_FILE")" && pwd)/$(basename "$NEXT_CONFIG_FILE")"
NEXT_CONFIG_NAME="$(basename "$NEXT_CONFIG_FILE" .json)"

# 04-prover-up.sh resolves configs by NAME from follow/configs or
# snapshot/configs — the next config must live in one of them.
if [[ ! -f "${PROJECT_ROOT}/configs/${NEXT_CONFIG_NAME}.json" \
   && ! -f "${PROJECT_ROOT}/../snapshot/configs/${NEXT_CONFIG_NAME}.json" ]]; then
    log_error "Next config must live in follow/configs/ or snapshot/configs/ (04-prover-up.sh resolves by name)"
    exit 1
fi

run_step() {  # in dry-run, print instead of executing
    if $DRY_RUN; then log_info "[dry-run] $*"; else "$@"; fi
}

pid_alive() {
    local pidfile="$1" pid
    [[ -f "$pidfile" ]] || return 1
    pid=$(cat "$pidfile" 2>/dev/null || true)
    [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

stop_component() {  # label pidfile cmdline-substring
    local label="$1" pidfile="$2" expect="$3" pid
    if [[ -f "$pidfile" ]]; then
        pid=$(cat "$pidfile" 2>/dev/null || true)
        if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
            if $DRY_RUN; then log_info "[dry-run] stop $label (pid $pid)"; return 0; fi
            # Coordinators run under setsid (process-group leaders); provers are
            # plain nohup children. Killing the group is safe in both cases.
            kill -- "-$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
            for _ in $(seq 1 10); do kill -0 "$pid" 2>/dev/null || break; sleep 1; done
            kill -9 -- "-$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
            log_ok "  stopped $label (pid $pid)"
            $DRY_RUN || rm -f "$pidfile"
            return 0
        fi
        # Stale pidfile — fall through to the pattern kill below.
        $DRY_RUN || rm -f "$pidfile"
    fi
    # Fallback: pattern kill (mirrors 11-follow-stop.sh)
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

# ─── Load configs ────────────────────────────────────────────────────────────
ANVIL_RPC=$(jq -r '.fork.anvil_rpc' "$CONFIG_FILE")
SHADOW_DSN=$(jq -r '.db.dsn' "$CONFIG_FILE")
SCROLL_CHAIN=$(jq -r '.contracts.scroll_chain' "$CONFIG_FILE")
MVRV=$(jq -r '.contracts.rollup_verifier' "$CONFIG_FILE")
GENESIS=$(jq -r '.genesis' "$CONFIG_FILE")
[[ "$GENESIS" = /* ]] || GENESIS="${REPO_ROOT}/${GENESIS}"
OLD_ASSETS=$(jq -r '.assets.assets_v2 // "coordinator/build/bin/assets_v2"' "$CONFIG_FILE")
[[ "$OLD_ASSETS" = /* ]] || OLD_ASSETS="${REPO_ROOT}/${OLD_ASSETS}"

NEW_S3=$(jq -r '.prover.s3_base_url' "$NEXT_CONFIG_FILE")
NEW_CIRCUIT_VERSION=$(jq -r '.prover.circuit_version' "$NEXT_CONFIG_FILE")
NEW_ASSETS=$(jq -r '.assets.assets_v2 // empty' "$NEXT_CONFIG_FILE")
if [[ -z "$NEW_ASSETS" ]]; then
    log_error "Next config must set .assets.assets_v2 (coordinator assets dir for the NEW release)"
    exit 1
fi
[[ "$NEW_ASSETS" = /* ]] || NEW_ASSETS="${REPO_ROOT}/${NEW_ASSETS}"

OLD_S3=$(jq -r '.prover.s3_base_url' "$CONFIG_FILE")

# ─── a. Sanity checks ────────────────────────────────────────────────────────
log_info "=== a. Sanity checks ==="
require_cmd jq; require_cmd cast; require_cmd psql; require_cmd curl; require_cmd forge

if [[ "$NEW_S3" == "$OLD_S3" ]]; then
    log_error "Next config has the same prover.s3_base_url as the current config ($NEW_S3) — nothing to upgrade"
    exit 1
fi
log_ok "  old release: $OLD_S3"
log_ok "  new release: $NEW_S3 (circuit_version $NEW_CIRCUIT_VERSION)"

pid_alive "${WORK_DIR}/anvil.pid" || log_warn "  anvil pidfile not alive — continuing anyway (RPC check below)"
cast block-number --rpc-url "$ANVIL_RPC" >/dev/null 2>&1 \
    || { log_error "Anvil RPC unreachable: $ANVIL_RPC"; exit 1; }
log_ok "  Anvil reachable"

psql "$SHADOW_DSN" -Atq -c "SELECT 1" >/dev/null || { log_error "shadow DB unreachable"; exit 1; }
log_ok "  shadow DB reachable"

PROVER_BIN="${REPO_ROOT}/target/release/prover"
[[ -x "$PROVER_BIN" ]] || { log_error "missing $PROVER_BIN — build the NEW prover first (cd zkvm-prover && make prover)"; exit 1; }
log_info "  prover binary: built $(stat -c '%y' "$PROVER_BIN" | cut -d. -f1), git $(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo '?')"
log_warn "  make sure the checked-out prover/coordinator binaries ARE the new version — this script does not rebuild them"

[[ -x "${COORD_DIR}/coordinator_api" ]]  || { log_error "missing ${COORD_DIR}/coordinator_api"; exit 1; }
[[ -x "${COORD_DIR}/coordinator_cron" ]] || { log_error "missing ${COORD_DIR}/coordinator_cron"; exit 1; }
[[ -f "${COORD_DIR}/conf/config.json" ]] || { log_error "missing ${COORD_DIR}/conf/config.json"; exit 1; }
log_ok "  coordinator binaries + config present"

# ─── b. Compute boundary batch N ─────────────────────────────────────────────
log_info "=== b. Compute upgrade boundary ==="
LAST_FINALIZED=$(cast call "$SCROLL_CHAIN" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$ANVIL_RPC" 2>/dev/null | awk '{print $1}' || true)
[[ -n "$LAST_FINALIZED" ]] || { log_error "failed to read lastFinalizedBatchIndex from $SCROLL_CHAIN on $ANVIL_RPC"; exit 1; }
log_info "  lastFinalizedBatchIndex on fork: $LAST_FINALIZED"

# P = newest batch covered by a proven-but-unfinalized bundle (old proofs to keep)
# U = oldest batch of an unproven pending bundle (must go to the new verifier)
P=$(psql "$SHADOW_DSN" -Atq -c "SELECT COALESCE(MAX(end_batch_index), -1) FROM bundle WHERE rollup_status <> 5 AND proving_status = 4;")
U=$(psql "$SHADOW_DSN" -Atq -c "SELECT COALESCE(MIN(end_batch_index), -1) FROM bundle WHERE rollup_status <> 5 AND proving_status <> 4;")
log_info "  proven pending up to batch:  $P"
log_info "  unproven pending from batch: $U"

CUTOVER_MODE=""
if [[ -n "$START_BATCH_OVERRIDE" ]]; then
    N="$START_BATCH_OVERRIDE"
    CUTOVER_MODE="explicit (--start-batch)"
    (( N > LAST_FINALIZED )) || { log_error "--start-batch $N must be > lastFinalized $LAST_FINALIZED"; exit 1; }
elif (( U == -1 )); then
    # Nothing unproven pending: boundary right after the finalized watermark;
    # only future bundles will exercise the new stack.
    N=$((LAST_FINALIZED + 1))
    CUTOVER_MODE="clean (no unproven pending bundles)"
elif (( P == -1 )); then
    N=$((LAST_FINALIZED + 1))
    CUTOVER_MODE="hard cutover (no proven pending bundles — everything re-proves)"
elif (( U > P )); then
    N="$U"
    CUTOVER_MODE="mixed (proven bundles <= $P keep old verifier, unproven >= $U re-prove)"
else
    N=$((LAST_FINALIZED + 1))
    CUTOVER_MODE="HARD CUTOVER FALLBACK (proven/unproven bundles interleave: P=$P U=$U — re-proving ALL pending)"
    log_warn "  no clean verifier boundary exists; falling back to hard cutover at $N"
fi
log_info "  boundary batch N = $N  [$CUTOVER_MODE]"

# ─── c. Record the OLD wrapper before touching the MVRV ─────────────────────
log_info "=== c. Current MVRV routing ==="
OLD_WRAPPER=$(cast call "$MVRV" "getVerifier(uint256,uint256)(address)" 10 "$((N - 1))" --rpc-url "$ANVIL_RPC" 2>/dev/null | tr -d ' ' || true)
if [[ -z "$OLD_WRAPPER" || "$OLD_WRAPPER" == "0x0000000000000000000000000000000000000000" ]]; then
    log_error "MVRV routes batch $((N - 1)) to no verifier — phase-1 setup is broken, aborting"
    exit 1
fi
log_ok "  old wrapper (batch $((N - 1))): $OLD_WRAPPER"

# ─── d. Stop provers + coordinators (Anvil/relayer/daemons keep running) ─────
log_info "=== d. Stop provers + coordinators ==="
for pf in "${WORK_DIR}"/prover-*/prover.pid; do
    [[ -f "$pf" ]] || continue
    gpu=$(basename "$(dirname "$pf")")
    stop_component "$gpu" "$pf" "prover.json"
done
if ! $DRY_RUN; then pkill -f "target/release/prover" 2>/dev/null || true; fi
stop_component "coordinator_cron" "${WORK_DIR}/coordinator-cron.pid" "coordinator_cron"
stop_component "coordinator_api"  "${WORK_DIR}/coordinator-api.pid"  "coordinator_api"

# ─── e. Reset in-flight tasks >= N; wipe prover proof caches ─────────────────
log_info "=== e. Reset task rows at/after batch $N ==="
RESET_SQL="
UPDATE bundle SET proving_status = 1, total_attempts = 0, active_attempts = 0, batch_proofs_status = 1
  WHERE rollup_status <> 5 AND end_batch_index >= ${N} AND proving_status <> 1;
UPDATE batch SET proving_status = 1, total_attempts = 0, active_attempts = 0, chunk_proofs_status = 0
  WHERE rollup_status <> 5 AND index >= ${N} AND proving_status <> 1;
-- Chunks: keep proofs ONLY for chunks of batches below N (they feed the
-- proven bundles we are keeping). Everything else — batches >= N and
-- not-yet-batched chunks — re-proves with the new circuits.
UPDATE chunk c SET proving_status = 1, total_attempts = 0, active_attempts = 0
  WHERE c.proving_status <> 1
    AND NOT EXISTS (
      SELECT 1 FROM batch b
      WHERE c.index BETWEEN b.start_chunk_index AND b.end_chunk_index
        AND b.index < ${N}
    );
"
if $DRY_RUN; then
    log_info "[dry-run] psql reset SQL:"; echo "$RESET_SQL"
else
    psql "$SHADOW_DSN" -Atq --set ON_ERROR_STOP=1 -c "$RESET_SQL" >/dev/null
    log_ok "  task rows >= batch $N reset to unassigned"
    # Trap 21: prover-local caches hold OLD-circuit proofs that would be
    # replayed as VData mismatches.
    rm -rf "${WORK_DIR}"/prover-*/db
    log_ok "  prover-local proof caches wiped"
fi

# ─── f. Download NEW verifier assets; point coordinator at them ──────────────
log_info "=== f. New verifier assets ==="
log_info "  downloading from ${NEW_S3}verifier/ -> $NEW_ASSETS"
if ! $DRY_RUN; then
    mkdir -p "$NEW_ASSETS"
    for f in verifier.bin root_verifier_vk openVmVk.json; do
        curl -fsSL "${NEW_S3}verifier/$f" -o "${NEW_ASSETS}/$f" \
            || { log_error "failed to download ${NEW_S3}verifier/$f (check the releases/ prefix, AGENTS.md S3 rules)"; exit 1; }
    done
    # Sanity: this must actually be a NEW release — identical chunk VKs mean
    # the S3 URL still points at the old circuits and the test would prove nothing.
    if [[ -f "${OLD_ASSETS}/openVmVk.json" ]]; then
        OLD_VK=$(jq -r '.chunk_vk' "${OLD_ASSETS}/openVmVk.json")
        NEW_VK=$(jq -r '.chunk_vk' "${NEW_ASSETS}/openVmVk.json")
        if [[ "$OLD_VK" == "$NEW_VK" ]]; then
            log_error "new assets have the SAME chunk_vk as the old ones — s3_base_url likely points at the old release"
            exit 1
        fi
        log_ok "  chunk_vk changed: ${OLD_VK:0:18}… -> ${NEW_VK:0:18}…"
    else
        log_warn "  old assets not found at $OLD_ASSETS — skipping VK-diff sanity check"
    fi
fi

COORD_CONF="${COORD_DIR}/conf/config.json"
log_info "  pointing coordinator $FORK_NAME verifier at $NEW_ASSETS"
if ! $DRY_RUN; then
    tmp=$(mktemp)
    jq --arg p "$NEW_ASSETS" --arg f "$FORK_NAME" \
        '(.prover_manager.verifier.verifiers[] | select(.fork_name == $f)).assets_path = $p' \
        "$COORD_CONF" > "$tmp" && mv "$tmp" "$COORD_CONF"
    log_ok "  $COORD_CONF updated"
fi

# ─── g. Restart coordinator_api + coordinator_cron ───────────────────────────
# Same launch commands as 10-follow-up.sh step f. OpenVM keygen takes ~2-3 min;
# provers started later retry until the API is up.
log_info "=== g. Restart coordinators ==="
if $DRY_RUN; then
    log_info "[dry-run] restart coordinator_api (:8390) + coordinator_cron (:8391)"
else
    (
        cd "$COORD_DIR"
        setsid nohup ./coordinator_api \
            --config conf/config.json --http --http.addr 0.0.0.0 \
            --http.port 8390 --service.port 8390 \
            >> "${WORK_DIR}/coordinator-api.log" 2>&1 &
        echo $! > "${WORK_DIR}/coordinator-api.pid"
    )
    sleep 2
    pid_alive "${WORK_DIR}/coordinator-api.pid" || { log_error "coordinator_api died; tail .work/coordinator-api.log"; exit 1; }
    (
        cd "$PROJECT_ROOT"
        setsid nohup "${COORD_DIR}/coordinator_cron" \
            --config "${PROJECT_ROOT}/configs/coordinator.json" \
            --genesis "$GENESIS" \
            --service.port 8391 \
            --log.file "${WORK_DIR}/coordinator-cron.log" --verbosity 3 \
            >> "${WORK_DIR}/coordinator-cron.log" 2>&1 &
        echo $! > "${WORK_DIR}/coordinator-cron.pid"
    )
    sleep 2
    pid_alive "${WORK_DIR}/coordinator-cron.pid" || { log_error "coordinator_cron died; tail .work/coordinator-cron.log"; exit 1; }
    log_ok "  coordinators restarting (OpenVM keygen takes ~2-3 min — prover logins retry)"
fi

# ─── h. Deploy + register the NEW wrapper (genuine updateVerifier path) ──────
log_info "=== h. Deploy + register new verifier ==="
run_step "${LIB_DIR}/03-deploy-verifier.sh" \
    --config "$NEXT_CONFIG_FILE" \
    --assets-dir "$NEW_ASSETS" \
    --start-batch "$N" \
    --genuine-register

# ─── i. Verify MVRV routing across the boundary ──────────────────────────────
log_info "=== i. Verify MVRV routing ==="
NEW_WRAPPER=$(grep '^WRAPPER_ADDR=' "${WORK_DIR}/verifier.env" 2>/dev/null | cut -d= -f2 || true)
if $DRY_RUN; then
    log_info "[dry-run] would check getVerifier(10, $((N - 1))) == old and getVerifier(10, $N) == new"
else
    [[ -n "$NEW_WRAPPER" ]] || { log_error "verifier.env missing WRAPPER_ADDR after 03-deploy-verifier.sh"; exit 1; }
    ROUTE_OLD=$(cast call "$MVRV" "getVerifier(uint256,uint256)(address)" 10 "$((N - 1))" --rpc-url "$ANVIL_RPC" 2>/dev/null | tr -d ' ' || true)
    ROUTE_NEW=$(cast call "$MVRV" "getVerifier(uint256,uint256)(address)" 10 "$N"       --rpc-url "$ANVIL_RPC" 2>/dev/null | tr -d ' ' || true)
    log_info "  getVerifier(10, $((N - 1))) = $ROUTE_OLD"
    log_info "  getVerifier(10, $N)       = $ROUTE_NEW"
    [[ "${ROUTE_OLD,,}" == "${OLD_WRAPPER,,}" ]] || { log_error "batch $((N - 1)) does NOT route to the old wrapper — legacy routing broken"; exit 1; }
    [[ "${ROUTE_NEW,,}" == "${NEW_WRAPPER,,}" ]] || { log_error "batch $N does NOT route to the new wrapper"; exit 1; }
    log_ok "  routing verified: old batches -> old wrapper, new batches -> new wrapper"
fi

# ─── j. Restart provers with the NEW circuits ────────────────────────────────
log_info "=== j. Restart provers ==="
run_step env GPUS="$GPUS" ASSETS_DIR="$NEW_ASSETS" \
    "${LIB_DIR}/04-prover-up.sh" --config "$NEXT_CONFIG_NAME"

# ─── k. Record the boundary in follow-run.env ────────────────────────────────
log_info "=== k. Record upgrade boundary ==="
FOLLOW_ENV="${WORK_DIR}/follow-run.env"
if $DRY_RUN; then
    log_info "[dry-run] would append UPGRADE_AT_BATCH=$N etc. to $FOLLOW_ENV"
else
    touch "$FOLLOW_ENV"
    tmp=$(mktemp)
    grep -v '^UPGRADE_' "$FOLLOW_ENV" > "$tmp" || true
    {
        cat "$tmp"
        echo "UPGRADE_AT_BATCH=${N}"
        echo "UPGRADE_AT_TIME=$(date -u +%Y-%m-%dT%H:%M:%S%z)"
        echo "UPGRADE_OLD_WRAPPER=${OLD_WRAPPER}"
        echo "UPGRADE_NEW_WRAPPER=${NEW_WRAPPER:-unknown}"
        echo "UPGRADE_CIRCUIT_VERSION=${NEW_CIRCUIT_VERSION}"
    } > "$FOLLOW_ENV"
    rm -f "$tmp"
    log_ok "  wrote boundary to $FOLLOW_ENV"
fi

echo ""
log_ok "Upgrade complete — finalization continues on the SAME fork and DB."
echo ""
echo "What to watch next:"
echo "  1. Bundles with end_batch < $N finalize via the OLD wrapper $OLD_WRAPPER"
echo "     (exercises MVRV legacy routing)."
echo "  2. The FIRST bundle with end_batch >= $N finalizes via the NEW wrapper"
echo "     ${NEW_WRAPPER:-<see verifier.env>} — the key acceptance moment."
echo "  3. make follow-status — finalized_lag should return to <= 1 after the"
echo "     re-proving burst; coordinator keygen idles provers for ~2-3 min."
echo "  4. make follow-report — splits metrics at UPGRADE_AT_BATCH."
