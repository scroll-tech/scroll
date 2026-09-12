#!/usr/bin/env bash
# 30-canary-upgrade.sh — t1 upgrade point of the CANARY parallel-upgrade test.
#
# Companion to `10-follow-up.sh --import-proofs` (Phase A). Unlike the
# hard-switch 20-upgrade.sh, this script resets NOTHING: bundles below the
# boundary N keep finalizing on imported production proofs through the OLD
# wrapper (MVRV legacy routing), while bundles at/after N are proven for the
# FIRST time by the new local stack and finalize through the NEW wrapper —
# old and new genuinely interleave on-chain.
#
#   a. sanity checks (next config differs, Anvil/DB reachable, binaries present,
#      remote_bundle_proof quarantine table exists)
#   b. compute boundary N = MIN(end_batch_index) of unfinalized bundles with NO
#      quarantined remote proof (fallback: lastFinalized+1) — everything < N is
#      finalizable from imported proofs, everything >= N goes to the new stack
#   c. record OLD wrapper = getVerifier(10, N-1) on the forked MVRV
#   d. download NEW verifier assets; point the coordinator config at them
#      (mirrors 20-upgrade.sh steps f/g, incl. the chunk_vk sanity check)
#   e. deploy + register the NEW wrapper via the GENUINE updateVerifier path
#      (03-deploy-verifier.sh --genuine-register --start-batch N)
#   f. verify MVRV routing: batch N-1 -> old wrapper, batch N -> new wrapper
#   g. write .work/canary.env PROOF_IMPORT_MAX_END_BATCH=N (poll-sync picks it
#      up next cycle — no daemon restart needed)
#   h. clear stale prover_task rows (Trap 34/35); NO bundle/batch/chunk reset,
#      NO prover-cache wipe (no local proving happened before t1)
#   i. start coordinator_api + coordinator_cron + provers on the NEW stack
#   j. record CANARY_* keys in .work/follow-run.env
#
# Idempotent: safe to re-run after 31-canary-rollback.sh (N is recomputed).
#
# Usage: ./30-canary-upgrade.sh --next-config configs/mainnet-next.json
#                               [--config configs/mainnet.json] [--start-batch N]
#                               [--docker-provers] [--dry-run]
#         --docker-provers: run provers as containers (PROVER_IMAGE, default
#                           scrolltech/prover:e2e-test) instead of bare metal

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
CANARY_ENV_FILE="${CANARY_ENV_FILE:-${WORK_DIR}/canary.env}"
COORD_DIR="${COORD_DIR:-${REPO_ROOT}/coordinator/build/bin}"
GPUS="${GPUS:-0,1}"
FORK_NAME="${FORK_NAME:-galileoV2}"
START_BATCH_OVERRIDE=""
DRY_RUN=false
DOCKER_PROVERS=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)      CONFIG_FILE="$2"; shift 2 ;;
        --next-config) NEXT_CONFIG_FILE="$2"; shift 2 ;;
        --start-batch) START_BATCH_OVERRIDE="$2"; shift 2 ;;
        --docker-provers) DOCKER_PROVERS=true; shift ;;
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

# Canary mode: the current config's s3_base_url is meaningless (Phase A never
# ran a local prover — old proofs come from the remote DB), so identical URLs
# are NOT an error here. The real "is this actually a different guest" check is
# the on-chain wrapper-digest comparison in step f.
if [[ "$NEW_S3" == "$OLD_S3" ]]; then
    log_info "  note: next config has the same s3_base_url as the current config ($NEW_S3) — fine in canary mode"
fi
log_ok "  new release: $NEW_S3 (circuit_version $NEW_CIRCUIT_VERSION)"

pid_alive "${WORK_DIR}/anvil.pid" || log_warn "  anvil pidfile not alive — continuing anyway (RPC check below)"
cast block-number --rpc-url "$ANVIL_RPC" >/dev/null 2>&1 \
    || { log_error "Anvil RPC unreachable: $ANVIL_RPC"; exit 1; }
log_ok "  Anvil reachable"

psql "$SHADOW_DSN" -Atq -c "SELECT 1" >/dev/null || { log_error "shadow DB unreachable"; exit 1; }
log_ok "  shadow DB reachable"

# Canary Phase A must have run with proof import — the quarantine table is
# created by sync-mainnet-db.py --import-proofs (10-follow-up.sh --import-proofs).
QUAR_COUNT=$(psql "$SHADOW_DSN" -Atq -c \
    "SELECT COUNT(*) FROM remote_bundle_proof" 2>/dev/null || echo "missing")
if [[ "$QUAR_COUNT" == "missing" ]]; then
    log_error "remote_bundle_proof table missing — bring the stack up with 10-follow-up.sh --import-proofs first"
    exit 1
fi
log_ok "  quarantine table remote_bundle_proof: $QUAR_COUNT proofs imported"

if $DOCKER_PROVERS; then
    PROVER_IMAGE="${PROVER_IMAGE:-scrolltech/prover:e2e-test}"
    docker image inspect "$PROVER_IMAGE" >/dev/null 2>&1 || {
        log_error "missing image $PROVER_IMAGE — build it first: docker build -f build/dockerfiles/prover.Dockerfile -t $PROVER_IMAGE ."
        exit 1
    }
    log_info "  prover image: $PROVER_IMAGE (docker mode)"
else
    PROVER_BIN="${REPO_ROOT}/target/release/prover"
    [[ -x "$PROVER_BIN" ]] || { log_error "missing $PROVER_BIN — build the NEW prover first (cd zkvm-prover && make prover)"; exit 1; }
    log_info "  prover binary: built $(stat -c '%y' "$PROVER_BIN" | cut -d. -f1), git $(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo '?')"
fi
log_warn "  make sure the checked-out prover/coordinator binaries (or PROVER_IMAGE) ARE the new version — this script does not rebuild them"

[[ -x "${COORD_DIR}/coordinator_api" ]]  || { log_error "missing ${COORD_DIR}/coordinator_api"; exit 1; }
[[ -x "${COORD_DIR}/coordinator_cron" ]] || { log_error "missing ${COORD_DIR}/coordinator_cron"; exit 1; }
[[ -f "${COORD_DIR}/conf/config.json" ]] || { log_error "missing ${COORD_DIR}/conf/config.json"; exit 1; }
# coordinator_api reads conf/genesis.json relative to its CWD at startup and
# CRITs without it (easy to miss on a fresh machine — the api dies seconds
# AFTER the pid check, and the provers then exhaust their login retries).
[[ -f "${COORD_DIR}/conf/genesis.json" ]] || { log_error "missing ${COORD_DIR}/conf/genesis.json (copy from tests/prover-e2e/mainnet-galileoV2/genesis.json)"; exit 1; }
log_ok "  coordinator binaries + config present"

# ─── b. Compute boundary batch N ─────────────────────────────────────────────
# N = oldest end_batch of an unfinalized bundle that has NO quarantined remote
# proof. Everything below N is finalizable from imported production proofs and
# stays on the old wrapper; everything at/after N is proven by the new stack.
log_info "=== b. Compute canary boundary ==="
LAST_FINALIZED=$(cast call "$SCROLL_CHAIN" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$ANVIL_RPC" 2>/dev/null | awk '{print $1}' || true)
[[ -n "$LAST_FINALIZED" ]] || { log_error "failed to read lastFinalizedBatchIndex from $SCROLL_CHAIN on $ANVIL_RPC"; exit 1; }
log_info "  lastFinalizedBatchIndex on fork: $LAST_FINALIZED"

U=$(psql "$SHADOW_DSN" -Atq -c \
    "SELECT COALESCE(MIN(end_batch_index), -1) FROM bundle
      WHERE rollup_status <> 5
        AND hash NOT IN (SELECT bundle_hash FROM remote_bundle_proof);")
log_info "  first unfinalized bundle without a remote proof ends at batch: $U"

# P = newest end_batch of an unfinalized bundle WITH a quarantined remote
# proof (the old-path backlog). N must be strictly greater than P, or such a
# bundle would route to the NEW wrapper while carrying an OLD proof and fail
# VerificationFailed. (The U=-1 shortcut MUST NOT fall back to
# lastFinalized+1 when a proven backlog exists.)
P=$(psql "$SHADOW_DSN" -Atq -c \
    "SELECT COALESCE(MAX(end_batch_index), -1) FROM bundle
      WHERE rollup_status <> 5
        AND hash IN (SELECT bundle_hash FROM remote_bundle_proof);")
log_info "  newest unfinalized bundle with a remote proof ends at batch: $P"

if [[ -n "$START_BATCH_OVERRIDE" ]]; then
    N="$START_BATCH_OVERRIDE"
    (( N > LAST_FINALIZED )) || { log_error "--start-batch $N must be > lastFinalized $LAST_FINALIZED"; exit 1; }
    (( N > P )) || { log_error "--start-batch $N must be > proven-backlog end $P (old proofs would route to the new wrapper)"; exit 1; }
    log_info "  boundary batch N = $N  [explicit --start-batch]"
elif (( U == -1 )); then
    N=$((P > LAST_FINALIZED ? P + 1 : LAST_FINALIZED + 1))
    log_info "  boundary batch N = $N  [no unproven frontier — only future bundles hit the new stack]"
else
    N=$(( U > P + 1 ? U : P + 1 ))
    log_info "  boundary batch N = $N  [remote-proof frontier]"
fi

# ─── c. Record the OLD wrapper before touching the MVRV ─────────────────────
log_info "=== c. Current MVRV routing ==="
OLD_WRAPPER=$(cast call "$MVRV" "getVerifier(uint256,uint256)(address)" 10 "$((N - 1))" --rpc-url "$ANVIL_RPC" 2>/dev/null | tr -d ' ' || true)
if [[ -z "$OLD_WRAPPER" || "$OLD_WRAPPER" == "0x0000000000000000000000000000000000000000" ]]; then
    log_error "MVRV routes batch $((N - 1)) to no verifier — phase-A setup is broken, aborting"
    exit 1
fi
log_ok "  old wrapper (batch $((N - 1))): $OLD_WRAPPER"

# ─── d. Download NEW verifier assets; point coordinator at them ──────────────
# Same as 20-upgrade.sh steps f/g. The coordinator is NOT running in Phase A,
# so the config rewrite below is race-free; if one IS running (unusual), stop
# it gracefully first so the restart in step i picks up the new assets.
log_info "=== d. New verifier assets ==="
if pid_alive "${WORK_DIR}/coordinator-api.pid" || pid_alive "${WORK_DIR}/coordinator-cron.pid"; then
    log_warn "  a coordinator is running (not expected in canary Phase A) — stopping it first"
    stop_component "coordinator_cron" "${WORK_DIR}/coordinator-cron.pid" "coordinator_cron"
    stop_component "coordinator_api"  "${WORK_DIR}/coordinator-api.pid"  "coordinator_api"
fi

log_info "  downloading from ${NEW_S3}verifier/ -> $NEW_ASSETS"
if ! $DRY_RUN; then
    mkdir -p "$NEW_ASSETS"
    for f in verifier.bin root_verifier_vk openVmVk.json; do
        curl -fsSL "${NEW_S3}verifier/$f" -o "${NEW_ASSETS}/$f" \
            || { log_error "failed to download ${NEW_S3}verifier/$f (check the releases/ prefix, AGENTS.md S3 rules)"; exit 1; }
    done
    # Trap 33: zkvm master >= bf887150 — the coordinator (libzkp universal
    # verifier) reads the BATCH circuit's agg_vk.bin as <assets>/agg_vk.bin.
    curl -fsSL "${NEW_S3}batch/agg_vk.bin" -o "${NEW_ASSETS}/agg_vk.bin" \
        || { log_error "failed to download ${NEW_S3}batch/agg_vk.bin (Trap 33: upload it to S3 after the guest build)"; exit 1; }
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

# ─── e. Deploy + register the NEW wrapper (genuine updateVerifier path) ──────
log_info "=== e. Deploy + register new verifier ==="
run_step "${LIB_DIR}/03-deploy-verifier.sh" \
    --config "$NEXT_CONFIG_FILE" \
    --assets-dir "$NEW_ASSETS" \
    --start-batch "$N" \
    --genuine-register

# ─── f. Verify MVRV routing across the boundary ──────────────────────────────
log_info "=== f. Verify MVRV routing ==="
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
    log_ok "  routing verified: batches < $N -> old wrapper, batches >= $N -> new wrapper"
    # Canary sanity: the NEW wrapper must bind DIFFERENT digests than the old
    # one — otherwise this "upgrade" swaps in the production guest and proves
    # nothing about the new stack.
    OLD_D1=$(cast call "$OLD_WRAPPER" "verifierDigest1()(bytes32)" --rpc-url "$ANVIL_RPC" 2>/dev/null || true)
    NEW_D1=$(cast call "$NEW_WRAPPER" "verifierDigest1()(bytes32)" --rpc-url "$ANVIL_RPC" 2>/dev/null || true)
    log_info "  digest1 old=$OLD_D1 new=$NEW_D1"
    if [[ -n "$OLD_D1" && -n "$NEW_D1" && "${OLD_D1,,}" == "${NEW_D1,,}" ]]; then
        log_error "new wrapper has the SAME verifierDigest1 as the production wrapper — not a real upgrade"
        exit 1
    fi
fi

# ─── g. Write the proof-import boundary ──────────────────────────────────────
# Next poll cycle onward, proofs for end_batch >= N stay quarantined in
# remote_bundle_proof (available for 31-canary-rollback.sh); only < N is
# applied to local bundle rows. No poll-sync restart needed.
log_info "=== g. Write proof-import boundary ==="
if $DRY_RUN; then
    log_info "[dry-run] would write PROOF_IMPORT_MAX_END_BATCH=$N to $CANARY_ENV_FILE"
else
    mkdir -p "$(dirname "$CANARY_ENV_FILE")"
    if [[ -f "$CANARY_ENV_FILE" ]]; then
        tmp=$(mktemp)
        grep -v '^PROOF_IMPORT_MAX_END_BATCH=' "$CANARY_ENV_FILE" > "$tmp" || true
        mv "$tmp" "$CANARY_ENV_FILE"
    fi
    echo "PROOF_IMPORT_MAX_END_BATCH=${N}" >> "$CANARY_ENV_FILE"
    log_ok "  wrote PROOF_IMPORT_MAX_END_BATCH=$N to $CANARY_ENV_FILE"
fi

# ─── h. Clear stale prover_task rows (NO bundle/batch/chunk reset) ───────────
# Unlike 20-upgrade.sh there is nothing local to re-prove: < N was never
# proven locally (imported proofs, marked proved), >= N is unproven and the
# new coordinator will pick it up naturally. Only in-flight/failed assignment
# rows could confuse the new coordinator (Trap 34/35) — delete those.
#
# Also mark chunks/batches of ALREADY-FINALIZED bundles as proved (NULL proof
# is safe — same pattern as 10-follow-up.sh step b: nothing consumes chunk/
# batch proofs once the bundle row carries one). Without this the new
# coordinator happily burns GPU re-proving ancient history whose bundles were
# finalized from imported proofs (observed: 50+ chunks of batches <= 519420
# dispatched at t1 before the manual fix).
log_info "=== h. Clear stale prover_task rows + skip finalized history ==="
SKIP_SQL="
DELETE FROM prover_task WHERE proving_status IN (1,3);
UPDATE chunk SET proving_status=4
  WHERE proving_status IN (1,2)
    AND batch_hash IN (SELECT hash FROM batch WHERE bundle_hash IN
        (SELECT hash FROM bundle WHERE rollup_status=5));
UPDATE batch SET proving_status=4
  WHERE proving_status IN (1,2)
    AND bundle_hash IN (SELECT hash FROM bundle WHERE rollup_status=5);
"
if $DRY_RUN; then
    log_info "[dry-run] psql: $SKIP_SQL"
else
    psql "$SHADOW_DSN" -Atq --set ON_ERROR_STOP=1 -c "$SKIP_SQL" >/dev/null
    log_ok "  stale prover_task rows deleted; finalized-history chunks/batches marked proved"
fi

# ─── i. Start the NEW coordinator + provers ──────────────────────────────────
# Same launch commands as 10-follow-up.sh step f / 20-upgrade.sh steps g+j.
# OpenVM keygen takes ~2-3 min; provers retry until the API is up.
log_info "=== i. Start new coordinator + provers ==="
if $DRY_RUN; then
    log_info "[dry-run] start coordinator_api (:8390) + coordinator_cron (:8391) + provers (GPUS=$GPUS, ASSETS_DIR=$NEW_ASSETS, config $NEXT_CONFIG_NAME)"
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
    log_ok "  coordinators starting (OpenVM keygen takes ~2-3 min — prover logins retry)"
fi
DOCKER_FLAG=()
$DOCKER_PROVERS && DOCKER_FLAG=(--docker)
run_step env GPUS="$GPUS" ASSETS_DIR="$NEW_ASSETS" PROVER_IMAGE="${PROVER_IMAGE:-scrolltech/prover:e2e-test}" \
    "${LIB_DIR}/04-prover-up.sh" --config "$NEXT_CONFIG_NAME" "${DOCKER_FLAG[@]}"

# ─── j. Record the boundary in follow-run.env ────────────────────────────────
log_info "=== j. Record canary boundary ==="
FOLLOW_ENV="${WORK_DIR}/follow-run.env"
if $DRY_RUN; then
    log_info "[dry-run] would append CANARY_AT_BATCH=$N etc. to $FOLLOW_ENV"
else
    touch "$FOLLOW_ENV"
    tmp=$(mktemp)
    # Strip only the keys this script manages — keep CANARY_MODE=1 from
    # 10-follow-up.sh --import-proofs.
    grep -vE '^CANARY_(AT_BATCH|AT_TIME|OLD_WRAPPER|NEW_WRAPPER|CIRCUIT_VERSION)=' "$FOLLOW_ENV" > "$tmp" || true
    {
        cat "$tmp"
        echo "CANARY_AT_BATCH=${N}"
        echo "CANARY_AT_TIME=$(date -u +%Y-%m-%dT%H:%M:%S%z)"
        echo "CANARY_OLD_WRAPPER=${OLD_WRAPPER}"
        echo "CANARY_NEW_WRAPPER=${NEW_WRAPPER:-unknown}"
        echo "CANARY_CIRCUIT_VERSION=${NEW_CIRCUIT_VERSION}"
    } > "$FOLLOW_ENV"
    rm -f "$tmp"
    log_ok "  wrote boundary to $FOLLOW_ENV"
fi

echo ""
log_ok "Canary upgrade complete — old (< $N, imported proofs) and new (>= $N,"
echo "   locally proven) bundles now finalize in parallel through their own wrappers."
echo ""
echo "What to watch next:"
echo "  1. Bundles with end_batch < $N keep finalizing via the OLD wrapper"
echo "     $OLD_WRAPPER from imported production proofs."
echo "  2. The FIRST bundle with end_batch >= $N finalizes via the NEW wrapper"
echo "     ${NEW_WRAPPER:-<see verifier.env>} — t2, the de-facto cutover."
echo "  3. Rollback drill: scripts/31-canary-rollback.sh (or make canary-rollback)"
echo "     restores the old wrapper at $N and applies quarantined proofs >= $N."
echo "  4. make follow-status / make follow-report — splits at CANARY_AT_BATCH."
