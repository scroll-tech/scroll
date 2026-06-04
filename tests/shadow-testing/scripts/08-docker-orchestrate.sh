#!/bin/bash
# One-command orchestrator for shadow fork testing.
# Usage: ./08-docker-orchestrate.sh [options]
#
# Options:
#   --bundle-range RANGE    Bundle index range, e.g. 17302:17305
#   --config NAME           Config name (mainnet|sepolia), default: mainnet
#   --phase PHASE           Phase to run: env|prove|finalize|all (default: all)
#   --skip-anvil-setup      Skip Anvil state setup (use existing state)
#   -h, --help              Show this help

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
source "${SCRIPT_DIR}/lib/anvil-utils.sh"

# ─── Defaults ────────────────────────────────────────────────────────────────
CONFIG="${CONFIG:-mainnet}"
export CONFIG
BUNDLE_RANGE=""
PHASE="all"
SKIP_ANVIL_SETUP=false

# ─── Parse args ──────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --bundle-range)   BUNDLE_RANGE="$2"; shift 2 ;;
        --config)         CONFIG="$2"; shift 2 ;;
        --phase)          PHASE="$2"; shift 2 ;;
        --skip-anvil-setup) SKIP_ANVIL_SETUP=true; shift ;;
        -h|--help)
            sed -n '2,14p' "$0"
            exit 0
            ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

if [[ -z "$BUNDLE_RANGE" ]]; then
    log_error "Must specify --bundle-range (e.g., 17302:17305)"
    exit 1
fi

BUNDLE_START="${BUNDLE_RANGE%%:*}"
BUNDLE_END="${BUNDLE_RANGE##*:}"

CONFIG_FILE="${SCRIPT_DIR}/../configs/${CONFIG}.json"
if [[ ! -f "$CONFIG_FILE" ]]; then
    log_error "Config file not found: $CONFIG_FILE"
    exit 1
fi

# ─── Read config ─────────────────────────────────────────────────────────────
log_info "Loading config: $CONFIG"

FORK_URL=$(jq -r '.fork.url' "$CONFIG_FILE")
FORK_BLOCK=$(jq -r '.fork.block_number' "$CONFIG_FILE")
ANVIL_RPC=$(jq -r '.fork.anvil_rpc' "$CONFIG_FILE")
DB_DSN=$(jq -r '.db.dsn' "$CONFIG_FILE")
PROD_DSN="${PROD_DSN:-postgresql://mainnet_infra_team_read_only:AuexDUuaarskbG6tr9CH9gXsJqp4at67mddAbMrt@localhost:15432/mainnet_rollup}"
SCROLL_CHAIN=$(jq -r '.contracts.scroll_chain' "$CONFIG_FILE")
L1_MSG_QUEUE=$(jq -r '.contracts.l1_message_queue_v2' "$CONFIG_FILE")
ROLLUP_VERIF=$(jq -r '.contracts.rollup_verifier' "$CONFIG_FILE")
DEPLOYED_VERIF=$(jq -r '.contracts.deployed_verifier' "$CONFIG_FILE")
OWNER=$(jq -r '.contracts.owner' "$CONFIG_FILE")
PROVER_EOA=$(jq -r '.accounts.prover_eoa' "$CONFIG_FILE")
COMMIT_EOA=$(jq -r '.accounts.commit_eoa' "$CONFIG_FILE")
LAST_FINALIZED=$(jq -r '.reset.last_finalized_batch_index' "$CONFIG_FILE")
NEXT_QUEUE=$(jq -r '.reset.next_unfinalized_queue_index' "$CONFIG_FILE")
CODEC_VERSION=$(jq -r '.reset.codec_version' "$CONFIG_FILE")
GENESIS=$(jq -r '.genesis' "$CONFIG_FILE")
L2_RPC=$(jq -r '.e2e.l2_rpc // .coordinator.l2geth // "https://mainnet-rpc.scroll.io"' "$CONFIG_FILE")
VALIDIUM_MODE=$(jq -r '.relayer.validium_mode // false' "$CONFIG_FILE")
CHAIN_MONITOR=$(jq -r '.relayer.chain_monitor_enabled // false' "$CONFIG_FILE")

# Docker compose file (run from repo root)
COMPOSE_FILE="${REPO_ROOT}/tests/shadow-testing/docker-compose.yml"

# ─── Helpers ─────────────────────────────────────────────────────────────────

compose() {
    docker compose -f "$COMPOSE_FILE" "$@"
}

wait_for_postgres() {
    local max=30
    local i=1
    while [ $i -le $max ]; do
        if compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then
            log_ok "PostgreSQL is ready"
            return 0
        fi
        log_info "Waiting for PostgreSQL... ($i/$max)"
        sleep 2
        ((i++))
    done
    log_error "PostgreSQL failed to start"
    return 1
}

wait_for_coordinator() {
    local max=60
    local i=1
    while [ $i -le $max ]; do
        if curl -s http://localhost:8390/ >/dev/null 2>&1; then
            log_ok "Coordinator is ready at :8390"
            return 0
        fi
        log_info "Waiting for coordinator... ($i/$max)"
        sleep 5
        ((i++))
    done
    log_error "Coordinator failed to start"
    return 1
}

wait_for_prover() {
    local max=120
    local i=1
    while [ $i -le $max ]; do
        # Check prover health via docker exec (no port mapping needed)
        if docker exec shadow-prover-gpu-0 sh -c "curl -sf http://localhost:10080/health >/dev/null 2>&1" 2>/dev/null; then
            log_ok "Prover is ready at :10080"
            return 0
        fi
        # Fallback: check if container is still running
        if ! docker ps --format '{{.Names}}' | grep -q "^shadow-prover-gpu-0$"; then
            log_error "Prover container exited unexpectedly"
            return 1
        fi
        log_info "Waiting for prover... ($i/$max)"
        sleep 10
        ((i++))
    done
    log_error "Prover failed to start"
    return 1
}

render_relayer_config() {
    local output="${SCRIPT_DIR}/../.work/relayer-${CONFIG}.json"
    mkdir -p "$(dirname "$output")"

    local template="${SCRIPT_DIR}/../configs/relayer.json.template"
    sed \
        -e "s|{{ANVIL_RPC}}|$ANVIL_RPC|g" \
        -e "s|{{DB_DSN}}|$DB_DSN|g" \
        -e "s|{{SCROLL_CHAIN}}|$SCROLL_CHAIN|g" \
        -e "s|{{L2_ENDPOINT}}|$L2_RPC|g" \
        -e "s|{{COMMIT_KEY}}|0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80|g" \
        -e "s|{{FINALIZE_KEY}}|0x01f1e12ee33f91d63172c3d51baa3cecb4469284b0ab45eed48e57fb5329ac4d|g" \
        -e "s|{{VALIDIUM_MODE}}|$VALIDIUM_MODE|g" \
        -e "s|{{CHAIN_MONITOR_ENABLED}}|$CHAIN_MONITOR|g" \
        "$template" > "$output"
    log_ok "Relayer config rendered: $output"
}

render_prover_config() {
    local gpu_id="${1:-0}"
    local output="${SCRIPT_DIR}/../.work/prover-${gpu_id}.json"
    mkdir -p "$(dirname "$output")"

    local prover_name
    prover_name=$(jq -r '.prover.name_prefix' "$CONFIG_FILE")
    local s3_url
    s3_url=$(jq -r '.prover.s3_base_url' "$CONFIG_FILE")

    cat > "$output" <<EOF
{
  "sdk_config": {
    "prover_name_prefix": "${prover_name}-${gpu_id}",
    "keys_dir": ".work",
    "coordinator": {
      "base_url": "http://shadow-coordinator:8390",
      "retry_count": 10,
      "retry_wait_time_sec": 10,
      "connection_timeout_sec": 1800
    },
    "prover": {
      "supported_proof_types": [1, 2, 3],
      "circuit_version": "v0.13.1"
    },
    "health_listener_addr": "0.0.0.0:10080",
    "db_path": ".work/db"
  },
  "circuits": {
    "galileoV2": {
      "base_url": "$s3_url",
      "workspace_path": ".work/galileo"
    }
  }
}
EOF
    log_ok "Prover config rendered: $output"
}

# ─── Phase: env ──────────────────────────────────────────────────────────────

run_env() {
    log_info "=== Phase: env ==="

    # 1. Check prerequisites
    log_info "Checking prerequisites..."
    command -v docker >/dev/null 2>&1 || { log_error "docker is required"; exit 1; }
    command -v jq >/dev/null 2>&1 || { log_error "jq is required"; exit 1; }
    command -v anvil >/dev/null 2>&1 || { log_error "anvil (Foundry) is required"; exit 1; }

    # 2. Start / reset PostgreSQL
    log_info "Starting PostgreSQL..."
    if compose ps postgres 2>/dev/null | grep -q "running"; then
        log_info "  PostgreSQL already running, stopping first..."
        compose stop postgres >/dev/null 2>&1 || true
        compose rm -f postgres >/dev/null 2>&1 || true
    fi
    compose up postgres -d
    wait_for_postgres

    # 3. Import bundle range
    log_info "Importing bundle range $BUNDLE_RANGE..."
    "${SCRIPT_DIR}/00-import-bundle-range.sh" \
        --bundle-range "$BUNDLE_RANGE" \
        --prod-dsn "$PROD_DSN" \
        --shadow-dsn "$DB_DSN"

    # 4. Start Anvil (bare metal)
    log_info "Starting Anvil fork..."
    local anvil_port="${ANVIL_RPC##*:}"
    local existing_pid
    existing_pid=$(lsof -ti :"$anvil_port" 2>/dev/null || true)
    if [[ -n "$existing_pid" ]]; then
        log_warn "Killing existing Anvil on port $anvil_port (PID $existing_pid)"
        kill "$existing_pid" 2>/dev/null || true
        sleep 2
    fi

    local state_file="${SCRIPT_DIR}/../.work/anvil-${CONFIG}.state.json"
    mkdir -p "$(dirname "$state_file")"

    nohup anvil \
        --fork-url "$FORK_URL" \
        --fork-block-number "$FORK_BLOCK" \
        --block-time 12 \
        --port "$anvil_port" \
        --host 0.0.0.0 \
        --state "$state_file" \
        >/dev/null 2>&1 &
    ANVIL_PID=$!
    log_info "Anvil started (PID $ANVIL_PID, port $anvil_port)"
    echo "$ANVIL_PID" > "${SCRIPT_DIR}/../.work/anvil-${CONFIG}.pid"
    sleep 3

    # 5. Setup Anvil state
    if [[ "$SKIP_ANVIL_SETUP" == "false" ]]; then
        log_info "Setting up Anvil state..."
        "${SCRIPT_DIR}/01-setup-anvil.sh" \
            --no-anvil \
            --anvil-rpc "$ANVIL_RPC" \
            --last-finalized "$LAST_FINALIZED" \
            --next-queue "$NEXT_QUEUE" \
            --deployed-verifier "$DEPLOYED_VERIF" \
            --prover-eoa "$PROVER_EOA" \
            --commit-eoa "$COMMIT_EOA" \
            --owner "$OWNER" \
            --scroll-chain "$SCROLL_CHAIN" \
            --l1-msg-queue "$L1_MSG_QUEUE" \
            --rollup-verifier "$ROLLUP_VERIF" \
            --db-dsn "$DB_DSN" \
            --codec-version "$CODEC_VERSION"
    else
        log_info "Skipping Anvil state setup (--skip-anvil-setup)"
    fi

    log_ok "Phase env complete!"
}

# ─── Phase: prove ────────────────────────────────────────────────────────────

run_prove() {
    log_info "=== Phase: prove ==="

    # 6. Start coordinator
    log_info "Starting coordinator..."
    compose up coordinator -d
    wait_for_coordinator

    # 7. Render prover config and start prover
    log_info "Rendering prover config..."
    render_prover_config 0
    log_info "Starting prover (GPU 0)..."
    compose --profile coordinator --profile prover up prover-gpu-0 -d
    wait_for_prover

    # 8. Wait for proofs
    log_info "Waiting for proofs..."
    "${SCRIPT_DIR}/05-wait-for-proofs.sh" \
        --db-dsn "$DB_DSN" \
        --bundle-range "$BUNDLE_RANGE"

    log_ok "Phase prove complete!"
}

# ─── Phase: finalize ─────────────────────────────────────────────────────────

run_finalize() {
    log_info "=== Phase: finalize ==="

    # 9. Render relayer config
    render_relayer_config

    # 10. Start relayer
    log_info "Starting relayer..."
    compose up relayer -d
    sleep 3

    # 11. Wait for finalization
    log_info "Waiting for finalization..."
    "${SCRIPT_DIR}/07-wait-for-finalize.sh" \
        --anvil-rpc "$ANVIL_RPC" \
        --scroll-chain "$SCROLL_CHAIN" \
        --bundle-range "$BUNDLE_RANGE" \
        --db-dsn "$DB_DSN"

    log_ok "Phase finalize complete!"
}

# ─── Main ────────────────────────────────────────────────────────────────────

log_info "Shadow Fork Orchestrator"
log_info "  Config:    $CONFIG"
log_info "  Bundles:   $BUNDLE_RANGE"
log_info "  Phase:     $PHASE"

# Ensure .work dir exists
mkdir -p "${SCRIPT_DIR}/../.work"

case "$PHASE" in
    env)
        run_env
        ;;
    prove)
        run_prove
        ;;
    finalize)
        run_finalize
        ;;
    all)
        run_env
        run_prove
        run_finalize
        log_info ""
        log_ok "🎉 Full pipeline complete!"
        log_info "   Config:  $CONFIG"
        log_info "   Bundles: $BUNDLE_RANGE"
        ;;
    *)
        log_error "Unknown phase: $PHASE (expected: env|prove|finalize|all)"
        exit 1
        ;;
esac
