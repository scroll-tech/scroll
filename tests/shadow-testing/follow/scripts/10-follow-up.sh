#!/usr/bin/env bash
# 10-follow-up.sh — one-shot orchestrator for "follow mode" shadow testing.
#
# Brings up the full follow-mode stack against the CURRENT mainnet tip:
#   a. sanity checks (postgres 5433, mainnet DSN, alchemy key, stale ports)
#   b. baseline DB sync from the mainnet read replica
#   c. Anvil fork at the current latest L1 block (via 01-setup-anvil.sh)
#   d. verifier wrapper deploy + registration (via 03-deploy-verifier.sh)
#   e. initial L1 queue rolling-hash sync (Trap 23)
#   f. coordinator_api + coordinator_cron
#   g. GPU provers (via 04-prover-up.sh)
#   h. rollup relayer (via 06-run-relayer.sh)
#   i. background daemon loops: poll sync / stale-proving sweeper / monitor
#   j. records run metadata in .work/follow-run.env
#
# Idempotent-ish: a component whose pidfile exists and is alive is skipped.
# Usage: ./10-follow-up.sh [--config configs/mainnet.json] [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
LIB_DIR="$(cd "${SCRIPT_DIR}/../../lib" && pwd)"
source "${LIB_DIR}/anvil-utils.sh"

# ─── Defaults (env-overridable) ──────────────────────────────────────────────
CONFIG_FILE="${CONFIG_FILE:-${PROJECT_ROOT}/configs/mainnet.json}"
# .work is shared by both modes and stays at tests/shadow-testing/.work.
WORK_DIR="${WORK_DIR:-${PROJECT_ROOT}/../.work}"
# Default mirrors sync-mainnet-db.py; override via env if the tunnel differs.
MAINNET_DSN="${MAINNET_DSN:-postgresql://mainnet_infra_team_read_only:AuexDUuaarskbG6tr9CH9gXsJqp4at67mddAbMrt@localhost:15432/mainnet_rollup}"
FOLLOW_RUN_HOURS="${FOLLOW_RUN_HOURS:-48}"
SYNC_POLL_INTERVAL="${SYNC_POLL_INTERVAL:-60}"
SWEEP_INTERVAL="${SWEEP_INTERVAL:-600}"
MONITOR_INTERVAL="${MONITOR_INTERVAL:-3600}"
EXPECTED_PROTOCOL_VERSION="${EXPECTED_PROTOCOL_VERSION:-10}"
# Follow mode runs on GalileoV2 (codec V10) blocks; the relayer must reject
# anything older, so force --min-codec-version 10 regardless of the config
# file's snapshot-mode value (see 06-run-relayer.sh MIN_CODEC override).
MIN_CODEC="${MIN_CODEC:-10}"
GPUS="${GPUS:-0,1}"
# Mainnet rule (AGENTS.md): resetting nextUnfinalizedQueueIndex to 0 is
# sufficient; sync-queue-hashes.py keeps nextCrossDomainMessageIndex current.
NEXT_QUEUE="${NEXT_QUEUE:-0}"
COORD_DIR="${COORD_DIR:-${REPO_ROOT}/coordinator/build/bin}"
DRY_RUN=false
RESET_DB=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)  CONFIG_FILE="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        --reset)   RESET_DB=true; shift ;;
        -h|--help) sed -n '2,19p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

mkdir -p "$WORK_DIR"

# ─── Helpers ─────────────────────────────────────────────────────────────────
pid_alive() {  # pidfile -> true if file exists and process alive
    local pidfile="$1" pid
    [[ -f "$pidfile" ]] || return 1
    pid=$(cat "$pidfile" 2>/dev/null || true)
    [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

any_of_our_pids() {  # pid -> true if one of our pidfiles references it
    local pid="$1" pf
    for pf in "$WORK_DIR"/*.pid "$WORK_DIR"/prover-*/prover.pid; do
        [[ -f "$pf" ]] || continue
        [[ "$(cat "$pf" 2>/dev/null)" == "$pid" ]] && return 0
    done
    return 1
}

check_port_free() {  # port name -> error if occupied by a process we don't track
    local port="$1" name="$2" pid
    pid=$(lsof -ti :"$port" 2>/dev/null | head -1 || true)
    if [[ -n "$pid" ]]; then
        if any_of_our_pids "$pid"; then
            log_info "  port $port ($name): held by tracked pid $pid (will skip restart)"
        elif $DRY_RUN; then
            log_warn "  port $port ($name): held by untracked pid $pid (would abort a real run)"
        else
            log_error "  port $port ($name): held by untracked pid $pid ($(tr '\0' ' ' < /proc/$pid/cmdline 2>/dev/null | cut -c1-80))"
            log_error "  Stop it first (scripts/11-follow-stop.sh, or kill manually)."
            exit 1
        fi
    else
        log_ok "  port $port ($name): free"
    fi
}

run_step() {  # in dry-run, print instead of executing
    if $DRY_RUN; then
        log_info "[dry-run] $*"
    else
        "$@"
    fi
}

# ─── Load config ─────────────────────────────────────────────────────────────
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
GENESIS=$(jq -r '.genesis' "$CONFIG_FILE")
[[ "$GENESIS" = /* ]] || GENESIS="${REPO_ROOT}/${GENESIS}"
CONFIG_NAME=$(basename "$CONFIG_FILE" .json)

# ─── a. Sanity checks ────────────────────────────────────────────────────────
log_info "=== a. Sanity checks ==="
require_cmd jq; require_cmd cast; require_cmd psql; require_cmd python3
require_cmd forge; require_cmd anvil; require_cmd lsof

# Alchemy key must be real, not a placeholder/demo
if [[ "$FORK_URL" == *demo* || "$FORK_URL" == *"<"* || "$FORK_URL" == *YOUR* ]]; then
    log_error "fork.url in $CONFIG_FILE has no real API key: $FORK_URL"
    exit 1
fi
log_ok "  fork.url key present"

log_info "  Checking shadow DB ($SHADOW_DSN) ..."
psql "$SHADOW_DSN" -Atq -c "SELECT 1" >/dev/null
log_ok "  shadow DB reachable"

# Attribute every write in the postgres server log (aids debugging "ghost"
# status writes — see TROUBLESHOOTING Trap 27 tail). ALTER SYSTEM persists in
# postgresql.auto.conf inside the data volume, so re-assert it on every
# bring-up: it is silently lost whenever the postgres container/volume is
# recreated.
if ! $DRY_RUN; then
    psql "$SHADOW_DSN" -Atq -c "ALTER SYSTEM SET log_statement = 'mod';" >/dev/null \
        && psql "$SHADOW_DSN" -Atq -c "SELECT pg_reload_conf();" >/dev/null \
        && log_ok "  postgres log_statement='mod' asserted (writes visible in the postgres server log)" \
        || log_warn "  failed to set log_statement='mod' (non-fatal)"
fi

log_info "  Checking mainnet read replica ..."
timeout 15 psql "$MAINNET_DSN" -Atq -c "SELECT 1" >/dev/null \
    || { log_error "mainnet DSN unreachable: $MAINNET_DSN (is the RDS tunnel up?)"; exit 1; }
log_ok "  mainnet DSN reachable"

check_port_free 8390 coordinator-api
check_port_free 8391 coordinator-cron
check_port_free 18545 anvil

# ─── b. Baseline DB sync ─────────────────────────────────────────────────────
# Window = mainnet's finalization lag, NOT history: L1 finalization is
# sequential (each finalize tx chains prevBatchHash), so the shadow must prove
# and finalize every committed-but-not-finalized bundle starting from the
# fork's lastFinalizedBatchIndex. Anything older is already finalized on the
# fork and is copied only as a parent reference (marked done post-baseline).
log_info "=== b. Baseline DB sync ==="
if $RESET_DB; then
    # --reset: wipe task tables so the DB contains ONLY the finalization-lag
    # window. Without this, stale rows from earlier runs/imports (thousands of
    # pending historical batches) are seen by the coordinator as work.
    #
    # CRITICAL (Trap 26): any component that writes task tables MUST be stopped
    # before the TRUNCATE. A relayer left running sees l2_block empty and its
    # l2 watcher starts a full L2 genesis crawl (block 1 → tip, days of RPC
    # calls, tens of GB of junk rows); an orphaned poll sync bulk-copies all
    # mainnet history (Trap 25). A full stop also resets this script's
    # idempotent pidfile checks, so every component is restarted fresh below.
    log_warn "  --reset: stopping all components before truncate"
    if ! $DRY_RUN; then
        bash "${SCRIPT_DIR}/11-follow-stop.sh" || log_warn "  11-follow-stop returned non-zero (continuing)"
    fi
    log_warn "  --reset: truncating chunk/batch/bundle/l2_block/l1_message"
    if ! $DRY_RUN; then
        psql "$SHADOW_DSN" -Atq --set ON_ERROR_STOP=1 -c "TRUNCATE chunk, batch, bundle, l2_block, l1_message" >/dev/null \
            || { log_error "TRUNCATE failed — aborting before baseline"; exit 1; }
        # Trap 21: wipe prover-local proof caches; they hold proofs from the
        # previous run/circuit version and get replayed as VData mismatches.
        rm -rf "${WORK_DIR}"/prover-*/db
        # Remove stale state from the previous fork too: 01/03 will re-create it.
        # follow-run.env carries the PREVIOUS run's report window (step j
        # preserves SHADOW_REPORT_START on re-entry) — drop it on --reset so
        # the new run gets a fresh report start time.
        rm -f "${WORK_DIR}/anvil-${CONFIG_NAME}.state.json" "${WORK_DIR}/verifier.env" "${WORK_DIR}/follow-run.env"
    fi
fi
LAST_FINALIZED=$(cast call "$SCROLL_CHAIN" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$FORK_URL" | awk '{print $1}')
MAINNET_TIP=$(psql "$MAINNET_DSN" -Atq -c "SELECT MAX(index) FROM batch" | tr -d ' ')
BASELINE_START=$((LAST_FINALIZED - 1))
log_info "  lastFinalizedBatchIndex: $LAST_FINALIZED; mainnet batch tip: $MAINNET_TIP"
log_info "  baseline window: ${BASELINE_START}..${MAINNET_TIP} ($((MAINNET_TIP - BASELINE_START)) batches = finalization lag)"
export MAINNET_DSN
run_step python3 "${SCRIPT_DIR}/sync-mainnet-db.py" --init-from-batch "$BASELINE_START"
# Boundary rows (<= lastFinalized) are already finalized on the fork: mark
# them done so the coordinator never wastes work proving them. Their proof
# columns stay NULL — safe, they are only referenced via row fields
# (parent state root / hash), never by proof bytes.
# Rows ABOVE the boundary may carry stale rollup_status from a previous run
# on an older fork (finalized there, but not on this fresh fork): reset them
# so the relayer re-commits/re-finalizes them in chain order. Proofs already
# generated stay valid (proof content depends on L2 data, not L1 state).
if ! $DRY_RUN; then
    MISC_RAW=$(cast call "$SCROLL_CHAIN" 0x06582acb --rpc-url "$FORK_URL")
    FORK_COMMITTED=$((16#${MISC_RAW:2:64}))
    log_info "  fork lastCommittedBatchIndex: $FORK_COMMITTED"
    psql "$SHADOW_DSN" -Atq -c "
        UPDATE batch  SET rollup_status = 5, proving_status = 4
          WHERE index <= ${LAST_FINALIZED} AND (rollup_status <> 5 OR proving_status <> 4);
        UPDATE bundle SET rollup_status = 5, proving_status = 4
          WHERE end_batch_index <= ${LAST_FINALIZED} AND (rollup_status <> 5 OR proving_status <> 4);
        UPDATE chunk  SET proving_status = 4 FROM batch b
          WHERE chunk.batch_hash = b.hash AND b.index <= ${LAST_FINALIZED} AND chunk.proving_status <> 4;
        UPDATE batch  SET rollup_status = 3, finalize_tx_hash = NULL, finalized_at = NULL
          WHERE index > ${LAST_FINALIZED} AND index <= ${FORK_COMMITTED} AND rollup_status <> 3;
        UPDATE batch  SET rollup_status = 1, commit_tx_hash = NULL, committed_at = NULL, finalize_tx_hash = NULL, finalized_at = NULL
          WHERE index > ${FORK_COMMITTED} AND rollup_status <> 1;
        UPDATE bundle SET rollup_status = 1, finalize_tx_hash = NULL, finalized_at = NULL
          WHERE end_batch_index > ${LAST_FINALIZED} AND rollup_status <> 1;
    " >/dev/null
    log_ok "  rollup_status aligned to fork boundaries (finalized<=${LAST_FINALIZED}, committed<=${FORK_COMMITTED})"
fi

# ─── c. Anvil fork at the CURRENT latest block ───────────────────────────────
log_info "=== c. Anvil fork at latest mainnet block ==="
# Resolve the block number explicitly (rather than passing "latest") so the
# fork point is logged and reproducible.
FORK_BLOCK_NUMBER=$(cast block-number --rpc-url "$FORK_URL")
log_info "  fork block:        $FORK_BLOCK_NUMBER"
log_info "  lastFinalized:     $LAST_FINALIZED (read from mainnet at fork block)"
log_info "  nextQueueIndex:    $NEXT_QUEUE"

STATE_FILE="${WORK_DIR}/anvil-${CONFIG_NAME}.state.json"
if [[ -f "$STATE_FILE" ]]; then
    # Anvil --state LOADS an existing file; a stale state file would silently
    # resurrect the previous fork. Back it up so this fork starts fresh.
    STATE_BAK="${STATE_FILE}.bak.$(date +%Y%m%d%H%M%S)"
    log_warn "  backing up existing state file -> $STATE_BAK"
    run_step mv "$STATE_FILE" "$STATE_BAK"
fi

ANVIL_FRESH=false
if pid_alive "${WORK_DIR}/anvil.pid"; then
    log_info "  Anvil already running (pid $(cat "${WORK_DIR}/anvil.pid")), skipping"
else
    ANVIL_FRESH=true
    # --deployed-verifier "" : 03-deploy-verifier.sh deploys a fresh wrapper
    # below; the copy-from-old-fork fallback in 01 cannot work on a new fork.
    run_step "${LIB_DIR}/01-setup-anvil.sh" \
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
fi

# ─── d. Deploy verifier wrapper ──────────────────────────────────────────────
# Full forge redeploy (not anvil_setCode) because anvil_setCode preserves the
# old wrapper's immutables (digest1/digest2/protocolVersion) — see AGENTS.md
# "anvil_setCode Does NOT Reset Immutables".
# EOA funding (owner/prover/commit) is already handled by 01-setup-anvil.sh
# step 7, so nothing extra is needed here.
log_info "=== d. Deploy verifier wrapper ==="
# Skip only when this is a re-entry into an already-running stack (Anvil not
# restarted by us) and a wrapper was previously deployed onto that same fork.
if [[ "$ANVIL_FRESH" != "true" && -s "${WORK_DIR}/verifier.env" && "${FORCE_REDEPLOY:-false}" != "true" ]]; then
    log_info "  verifier.env present and Anvil unchanged, reusing $(grep '^WRAPPER_ADDR=' "${WORK_DIR}/verifier.env" | cut -d= -f2)"
else
    run_step "${LIB_DIR}/03-deploy-verifier.sh" --config "$CONFIG_FILE"
fi

# ─── e. Initial queue-hash sync (Trap 23) ────────────────────────────────────
log_info "=== e. L1 queue rolling-hash sync (Trap 23) ==="
export FORK_RPC="$ANVIL_RPC"
run_step python3 "${LIB_DIR}/sync-queue-hashes.py"

# ─── f. Coordinator api + cron ───────────────────────────────────────────────
log_info "=== f. Coordinator ==="
[[ -x "${COORD_DIR}/coordinator_api" ]] || { log_error "missing ${COORD_DIR}/coordinator_api (build: cd coordinator && make coordinator_api)"; exit 1; }
[[ -x "${COORD_DIR}/coordinator_cron" ]] || { log_error "missing ${COORD_DIR}/coordinator_cron"; exit 1; }

# coordinator_api keeps using coordinator/build/bin/conf/config.json: verified
# its db.dsn already points at the shadow DB (localhost:5433/shadow_rollup) and
# it carries the api-specific prover_manager/verifier sections the running
# setup relies on. The shadow-testing follow/configs/coordinator.json is only needed
# for coordinator_cron, whose timeout checker requires the correct
# bundle/batch/chunk collection_time_sec values.
if pid_alive "${WORK_DIR}/coordinator-api.pid"; then
    log_info "  coordinator_api already running (pid $(cat "${WORK_DIR}/coordinator-api.pid")), skipping"
else
    log_info "  starting coordinator_api (log: .work/coordinator-api.log)"
    if ! $DRY_RUN; then
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
    fi
fi

# coordinator_cron = timeout checker; must use the shadow-testing config with
# the correct collection times, plus the GalileoV2 genesis (current practice).
if pid_alive "${WORK_DIR}/coordinator-cron.pid"; then
    log_info "  coordinator_cron already running (pid $(cat "${WORK_DIR}/coordinator-cron.pid")), skipping"
else
    log_info "  starting coordinator_cron (log: .work/coordinator-cron.log)"
    if ! $DRY_RUN; then
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
    fi
fi

# ─── g. Provers ──────────────────────────────────────────────────────────────
log_info "=== g. GPU provers ==="
PROVERS_RUNNING=true
IFS=',' read -ra GPU_ARRAY <<< "$GPUS"
for gpu in "${GPU_ARRAY[@]}"; do
    pid_alive "${WORK_DIR}/prover-${gpu}/prover.pid" || PROVERS_RUNNING=false
done
if $PROVERS_RUNNING; then
    log_info "  provers on GPUs $GPUS already running, skipping"
else
    run_step env GPUS="$GPUS" "${LIB_DIR}/04-prover-up.sh" --config "$CONFIG_NAME"
fi

# ─── h. Relayer ──────────────────────────────────────────────────────────────
log_info "=== h. Rollup relayer ==="
# Trap 27: always (re-)ensure the relayer's senders can transact on the fork.
# 01-setup-anvil.sh does this too, but the idempotent path skips it when Anvil
# is already running (e.g. after an Anvil state restore wiped balances,
# sequencer authorization, or account nonces). All three ops are idempotent.
# Keep these keys in sync with 06-run-relayer.sh.
COMMIT_KEY="0x0afd95b5f1d9ef456b33c4e3720fbe70de7b4ff6e868fef454dc0aa60b09d8dc"
FINALIZE_KEY="0x01f1e12ee33f91d63172c3d51baa3cecb4469284b0ab45eed48e57fb5329ac4d"
COMMIT_SENDER=$(cast wallet address "$COMMIT_KEY")
FINALIZE_SENDER=$(cast wallet address "$FINALIZE_KEY")
if ! $DRY_RUN; then
    cast rpc anvil_setBalance "$COMMIT_SENDER" 0x56bc75e2d63100000 --rpc-url "$ANVIL_RPC" >/dev/null
    cast rpc anvil_setBalance "$FINALIZE_SENDER" 0x56bc75e2d63100000 --rpc-url "$ANVIL_RPC" >/dev/null
    if [[ "$(cast call "$SCROLL_CHAIN" "isSequencer(address)(bool)" "$COMMIT_SENDER" --rpc-url "$ANVIL_RPC" 2>/dev/null)" != "true" ]]; then
        log_warn "  authorizing $COMMIT_SENDER as sequencer on the fork"
        cast rpc anvil_impersonateAccount "$OWNER" --rpc-url "$ANVIL_RPC" >/dev/null
        cast send --gas-limit 5000000 "$SCROLL_CHAIN" "addSequencer(address)" "$COMMIT_SENDER" \
            --from "$OWNER" --rpc-url "$ANVIL_RPC" --unlocked >/dev/null \
            || { cast rpc anvil_stopImpersonatingAccount "$OWNER" --rpc-url "$ANVIL_RPC" >/dev/null; log_error "addSequencer failed"; exit 1; }
        cast rpc anvil_stopImpersonatingAccount "$OWNER" --rpc-url "$ANVIL_RPC" >/dev/null
    fi
    log_ok "  relayer EOAs funded (100 ETH), commit sender is sequencer"
fi
if pid_alive "${WORK_DIR}/relayer-${CONFIG_NAME}.pid"; then
    log_info "  relayer already running (pid $(cat "${WORK_DIR}/relayer-${CONFIG_NAME}.pid")), skipping"
else
    run_step env MIN_CODEC="$MIN_CODEC" "${LIB_DIR}/06-run-relayer.sh" --config "$CONFIG_NAME"
fi

# ─── i. Background daemon loops ──────────────────────────────────────────────
log_info "=== i. Daemon loops ==="
start_loop() {  # name interval logfile command...
    local name="$1" interval="$2" logfile="$3"; shift 3
    local pidfile="${WORK_DIR}/${name}.pid"
    if pid_alive "$pidfile"; then
        log_info "  ${name} loop already running (pid $(cat "$pidfile")), skipping"
        return 0
    fi
    log_info "  starting ${name} loop (every ${interval}s, log: ${logfile})"
    if ! $DRY_RUN; then
        # Shell-escape each arg into a single command string. (Naive \"$@\"
        # inside double quotes collapses all args into ONE quoted word.)
        local cmd
        cmd=$(printf '%q ' "$@")
        # Detached self-restarting loop; pidfile tracks the loop shell.
        setsid nohup bash -c "while true; do ${cmd} >>'${logfile}' 2>&1 || true; sleep ${interval}; done" \
            >/dev/null 2>&1 &
        echo $! > "$pidfile"
    fi
}

start_loop sync-mainnet-db "$SYNC_POLL_INTERVAL" "${WORK_DIR}/sync-mainnet-db.log" \
    python3 "${SCRIPT_DIR}/sync-mainnet-db.py" --poll-interval "$SYNC_POLL_INTERVAL"
# Sweeper resets proving_status AND total_attempts (Trap 22/23 starvation traps).
start_loop sweeper "$SWEEP_INTERVAL" "${WORK_DIR}/sweeper.log" \
    bash "${SCRIPT_DIR}/sweep-stale-proving.sh"
# monitor-catchup.py appends to .work/catchup-metrics.log itself.
start_loop monitor "$MONITOR_INTERVAL" "${WORK_DIR}/monitor.log" \
    python3 "${SCRIPT_DIR}/monitor-catchup.py"

# ─── j. Run metadata + summary ───────────────────────────────────────────────
log_info "=== j. Run metadata ==="
FOLLOW_ENV="${WORK_DIR}/follow-run.env"
if ! $DRY_RUN; then
    # Preserve an existing SHADOW_REPORT_START when components were skipped
    # (i.e. this is a re-entry into an already-running run, not a fresh run).
    EXISTING_START=""
    [[ -f "$FOLLOW_ENV" ]] && EXISTING_START=$(grep '^SHADOW_REPORT_START=' "$FOLLOW_ENV" | cut -d= -f2- || true)
    REPORT_START="${EXISTING_START:-$(date -u +%Y-%m-%dT%H:%M:%S%z)}"
    cat > "$FOLLOW_ENV" <<EOF
SHADOW_REPORT_START=${REPORT_START}
FOLLOW_RUN_HOURS=${FOLLOW_RUN_HOURS}
EXPECTED_PROTOCOL_VERSION=${EXPECTED_PROTOCOL_VERSION}
FORK_BLOCK=${FORK_BLOCK_NUMBER}
BASELINE_START_BATCH=${BASELINE_START}
EOF
    log_ok "  wrote $FOLLOW_ENV (SHADOW_REPORT_START=$REPORT_START, FOLLOW_RUN_HOURS=$FOLLOW_RUN_HOURS)"
fi

echo ""
log_ok "Follow-mode stack is up. Running components:"
for comp in anvil coordinator-api coordinator-cron relayer-${CONFIG_NAME} sync-mainnet-db sweeper monitor; do
    pf="${WORK_DIR}/${comp}.pid"
    if pid_alive "$pf"; then echo "  - $comp (pid $(cat "$pf"))"; fi
done
for gpu in "${GPU_ARRAY[@]}"; do
    pf="${WORK_DIR}/prover-${gpu}/prover.pid"
    if pid_alive "$pf"; then echo "  - prover-gpu${gpu} (pid $(cat "$pf"))"; fi
done
echo ""
echo "Useful commands:"
echo "  make follow-status    # latest monitor snapshot + finalize lag"
echo "  make follow-report    # throughput report for this run"
echo "  make re-fork          # re-fork Anvil at latest block (recovery)"
echo "  make follow-stop      # stop everything (keeps postgres)"
echo "  scripts/11-follow-stop.sh --keep-anvil   # stop all but Anvil+DB (run from follow/)"
