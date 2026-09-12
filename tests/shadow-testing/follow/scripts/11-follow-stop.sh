#!/usr/bin/env bash
# 11-follow-stop.sh — stop everything started by 10-follow-up.sh.
#
# Stops: daemon loops (sync/sweeper/monitor), relayer, provers,
# coordinator_cron, coordinator_api, and Anvil (unless --keep-anvil).
# Postgres (shadow DB) is never touched. Cleans up pidfiles afterwards.
#
# Usage: ./11-follow-stop.sh [--keep-anvil] [--config configs/mainnet.json]

set -uo pipefail  # no -e: a single dead pidfile must not abort the cleanup

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/../../lib/anvil-utils.sh"
# anvil-utils.sh re-enables -e (set -euo pipefail at its top) — undo that:
# this script's contract is best-effort cleanup that never aborts midway.
set +e

CONFIG_FILE="${CONFIG_FILE:-${PROJECT_ROOT}/configs/mainnet.json}"
# .work is shared by both modes and stays at tests/shadow-testing/.work.
WORK_DIR="${WORK_DIR:-${PROJECT_ROOT}/../.work}"
KEEP_ANVIL=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --keep-anvil) KEEP_ANVIL=true; shift ;;
        --config)     CONFIG_FILE="$2"; shift 2 ;;
        -h|--help)    sed -n '2,9p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

[[ -f "$CONFIG_FILE" ]] && CONFIG_NAME=$(basename "$CONFIG_FILE" .json) || CONFIG_NAME="mainnet"

STOPPED=()

# stop_pid <label> <pidfile> <cmdline-substring>
# Kills the pid only if its cmdline matches the expected substring (guards
# against stale pidfiles whose pid was reused by an unrelated process).
stop_pid() {
    local label="$1" pidfile="$2" expect="$3" pid
    [[ -f "$pidfile" ]] || return 1
    pid=$(cat "$pidfile" 2>/dev/null || true)
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
        if ! tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null | grep -qF "$expect"; then
            log_warn "  $label: pidfile stale (pid $pid is not '$expect'), not killing"
            rm -f "$pidfile"
            return 2  # let caller try a pattern/port fallback
        else
            # Loops/daemons are started via setsid, so the pidfile pid is a
            # process-group leader: kill the whole group, otherwise children
            # (e.g. the python sync process inside a bash loop) are orphaned
            # and keep running against the DB.
            kill -- "-$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
            for _ in $(seq 1 10); do kill -0 "$pid" 2>/dev/null || break; sleep 1; done
            if kill -0 "$pid" 2>/dev/null; then
                log_warn "  $label (pid $pid) ignored SIGTERM, sending SIGKILL"
                kill -9 -- "-$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
            fi
            STOPPED+=("$label (pid $pid)")
        fi
    fi
    rm -f "$pidfile"
    return 0
}

# fallback_pkill <label> <pattern> — used when no valid pidfile was found
fallback_pkill() {
    local label="$1" pattern="$2" pids
    pids=$(pgrep -f "$pattern" 2>/dev/null || true)
    if [[ -n "$pids" ]]; then
        log_warn "  $label: no pidfile, killing by pattern '$pattern' (pids: $(echo $pids | tr '\n' ' '))"
        echo "$pids" | xargs -r kill 2>/dev/null || true
        sleep 2
        pgrep -f "$pattern" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
        STOPPED+=("$label (pattern)")
    fi
}

log_info "=== Stopping follow-mode stack ==="

# 1. Daemon loops first, so they don't relaunch children during teardown.
stop_pid "sync loop"      "${WORK_DIR}/sync-mainnet-db.pid" "sync-mainnet-db.py" \
    || fallback_pkill "sync loop" "sync-mainnet-db.py --poll-interval"
stop_pid "sweeper loop"   "${WORK_DIR}/sweeper.pid" "sweep-stale-proving" || true
stop_pid "monitor loop"   "${WORK_DIR}/monitor.pid" "monitor-catchup.py" \
    || fallback_pkill "monitor loop" "monitor-catchup.py"

# 2. Relayer
stop_pid "relayer" "${WORK_DIR}/relayer-${CONFIG_NAME}.pid" "rollup_relayer" \
    || fallback_pkill "relayer" "rollup_relayer.*relayer-${CONFIG_NAME}"

# 3. Provers (per-GPU pidfiles written by 04-prover-up.sh)
# The expect pattern is the config path ("prover.json"), which matches both
# bare-metal (target/release/prover --config …/prover.json) and docker-mode
# (prover --config …/prover.json) cmdlines.
FOUND_PROVER=false
for pf in "${WORK_DIR}"/prover-*/prover.pid; do
    [[ -f "$pf" ]] || continue
    FOUND_PROVER=true
    gpu=$(basename "$(dirname "$pf")")
    stop_pid "$gpu" "$pf" "prover.json"
done
if [[ "$FOUND_PROVER" == "false" && -f "$CONFIG_FILE" ]]; then
    PROVER_PREFIX=$(jq -r '.prover.name_prefix // empty' "$CONFIG_FILE" 2>/dev/null)
    [[ -n "$PROVER_PREFIX" ]] && fallback_pkill "provers" "prover.*${PROVER_PREFIX}"
fi
# Remove legacy top-level prover pidfiles if present
rm -f "${WORK_DIR}"/prover-[0-9]*.pid
# Docker-mode provers (04-prover-up.sh --docker): the pidfile kill above
# normally stops them (host PID recorded), but make sure no container lingers.
if command -v docker >/dev/null 2>&1 && docker ps -aq --filter 'name=^shadow-prover-' 2>/dev/null | grep -q .; then
    docker rm -f $(docker ps -aq --filter 'name=^shadow-prover-') >/dev/null 2>&1 || true
    log_info "  removed shadow-prover-* containers"
fi

# 4. Coordinators
stop_pid "coordinator_cron" "${WORK_DIR}/coordinator-cron.pid" "coordinator_cron" \
    || fallback_pkill "coordinator_cron" "coordinator_cron --config"
stop_pid "coordinator_api"  "${WORK_DIR}/coordinator-api.pid" "coordinator_api" \
    || fallback_pkill "coordinator_api" "coordinator_api --config"
rm -f "${WORK_DIR}/coordinator.pid"  # legacy
for port in 8391 8390; do
    pid=$(lsof -ti :"$port" 2>/dev/null | head -1 || true)
    if [[ -n "$pid" ]] && tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null | grep -qF "coordinator"; then
        log_warn "  port $port still held by coordinator pid $pid, killing"
        kill "$pid" 2>/dev/null || true
        STOPPED+=("coordinator on :$port (pid $pid)")
    fi
done

# 5. Anvil (optional)
if $KEEP_ANVIL; then
    log_info "  --keep-anvil: leaving Anvil and shadow DB running"
else
    if ! stop_pid "anvil" "${WORK_DIR}/anvil.pid" "anvil"; then
        pid=$(lsof -ti :18545 2>/dev/null | head -1 || true)
        if [[ -n "$pid" ]] && tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null | grep -qF "anvil"; then
            log_warn "  anvil: no pidfile, killing pid $pid on port 18545"
            kill "$pid" 2>/dev/null || true
            STOPPED+=("anvil (pid $pid)")
        fi
    fi
fi

echo ""
if [[ ${#STOPPED[@]} -eq 0 ]]; then
    log_info "Nothing was running."
else
    log_ok "Stopped:"
    printf '  - %s\n' "${STOPPED[@]}"
fi
$KEEP_ANVIL && log_info "Anvil + shadow DB left running (restart stack with scripts/10-follow-up.sh)."
exit 0
