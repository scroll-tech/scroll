#!/usr/bin/env bash
# prove-only-up.sh — bring up the prove-only follow stack (no anvil, no relayer):
# sync loop + sweeper + coordinator_api/cron + docker prover (GPU 1).
# Proofs for the live mainnet frontier are generated locally and stored in the
# shadow DB. Stop with tests/shadow-testing/follow/scripts/11-follow-stop.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FOLLOW_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${FOLLOW_DIR}/../../.." && pwd)"
WORK_DIR="${REPO_ROOT}/tests/shadow-testing/.work"
COORD_DIR="${REPO_ROOT}/coordinator/build/bin"
export MAINNET_DSN='postgresql://mainnet_infra_team_read_only:AuexDUuaarskbG6tr9CH9gXsJqp4at67mddAbMrt@192.168.1.108:15432/mainnet_rollup'
export SHADOW_DSN='postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup'

start_loop() { # name interval logfile cmd...
    local name="$1" interval="$2" logfile="$3"; shift 3
    local pidfile="${WORK_DIR}/${name}.pid"
    if [[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
        echo "  $name loop already running (pid $(cat "$pidfile")), skipping"
        return 0
    fi
    local cmd
    cmd=$(printf '%q ' "$@")
    setsid nohup bash -c "while true; do ${cmd} >>'${logfile}' 2>&1 || true; sleep ${interval}; done" >/dev/null 2>&1 &
    echo $! > "$pidfile"
    echo "  started $name loop (pid $(cat "$pidfile"))"
}

echo "=== sync + sweeper loops ==="
start_loop sync-mainnet-db 60 "${WORK_DIR}/sync-mainnet-db.log" \
    python3 "${FOLLOW_DIR}/scripts/sync-mainnet-db.py" --poll-interval 60
start_loop sweeper 600 "${WORK_DIR}/sweeper.log" \
    bash "${FOLLOW_DIR}/scripts/sweep-stale-proving.sh"

if [[ "${1:-}" == "--sync-only" ]]; then
    echo "sync-only mode: coordinator/prover NOT started (catch-up phase)"
    exit 0
fi

echo "=== coordinator_api + coordinator_cron ==="
if [[ -f "${WORK_DIR}/coordinator-api.pid" ]] && kill -0 "$(cat "${WORK_DIR}/coordinator-api.pid")" 2>/dev/null; then
    echo "  coordinator_api already running"
else
    ( cd "$COORD_DIR" && setsid nohup ./coordinator_api \
        --config conf/config.json --http --http.addr 0.0.0.0 \
        --http.port 8390 --service.port 8390 \
        >> "${WORK_DIR}/coordinator-api.log" 2>&1 & echo $! > "${WORK_DIR}/coordinator-api.pid" )
    echo "  coordinator_api started (pid $(cat "${WORK_DIR}/coordinator-api.pid"))"
fi
if [[ -f "${WORK_DIR}/coordinator-cron.pid" ]] && kill -0 "$(cat "${WORK_DIR}/coordinator-cron.pid")" 2>/dev/null; then
    echo "  coordinator_cron already running"
else
    ( cd "$FOLLOW_DIR" && setsid nohup "${COORD_DIR}/coordinator_cron" \
        --config "${FOLLOW_DIR}/configs/coordinator.json" \
        --genesis "${REPO_ROOT}/tests/prover-e2e/mainnet-galileoV2/genesis.json" \
        --service.port 8391 \
        --log.file "${WORK_DIR}/coordinator-cron.log" --verbosity 3 \
        >> "${WORK_DIR}/coordinator-cron.log" 2>&1 & echo $! > "${WORK_DIR}/coordinator-cron.pid" )
    echo "  coordinator_cron started (pid $(cat "${WORK_DIR}/coordinator-cron.pid"))"
fi

echo "=== docker prover (GPU 1) ==="
GPUS=1 ASSETS_DIR="${COORD_DIR}/assets_v2_next" \
    PROVER_IMAGE="${PROVER_IMAGE:-scrolltech/cuda-prover:v4.8.0-8cc4b0dd-bf88715-}" \
    "${REPO_ROOT}/tests/shadow-testing/lib/04-prover-up.sh" --config mainnet-next --docker

echo "prove-only stack is up"
