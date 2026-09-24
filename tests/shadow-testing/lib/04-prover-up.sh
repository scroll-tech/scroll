#!/usr/bin/env bash
# Start prover(s) for shadow testing
# Usage: ./04-prover-up.sh --config mainnet --bundle-range 17297:17301 [--gpus 0,1]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/anvil-utils.sh"

CONFIG="${CONFIG:-mainnet}"
GPUS="${GPUS:-0,1}"
DOCKER=false
# Docker mode: image must already exist locally (built via
# build/dockerfiles/prover.Dockerfile from THIS checkout). The devops
# prover-image-build workflow produces the same packaging in CI.
PROVER_IMAGE="${PROVER_IMAGE:-scrolltech/prover:e2e-test}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)      CONFIG="$2"; shift 2 ;;
        --gpus)        GPUS="$2"; shift 2 ;;
        --docker)      DOCKER=true; shift ;;
        -h|--help)     sed -n '2,5p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

# lib/ is shared by both modes: resolve the config from whichever mode dir has it.
CONFIG_FILE=""
for d in "${SCRIPT_DIR}/../follow/configs" "${SCRIPT_DIR}/../snapshot/configs"; do
    if [[ -f "$d/${CONFIG}.json" ]]; then CONFIG_FILE="$d/${CONFIG}.json"; break; fi
done
if [[ -z "$CONFIG_FILE" ]]; then
    log_error "Config not found: follow/configs/${CONFIG}.json or snapshot/configs/${CONFIG}.json"
    exit 1
fi

# Read config
PROVER_NAME=$(jq -r '.prover.name_prefix' "$CONFIG_FILE")
CIRCUIT_VERSION=$(jq -r '.prover.circuit_version' "$CONFIG_FILE")
S3_URL=$(jq -r '.prover.s3_base_url' "$CONFIG_FILE")
DB_DSN=$(jq -r '.db.dsn' "$CONFIG_FILE")

REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Load circuit VKs from coordinator assets so we can set per-circuit S3 detours.
# v0.9.0 stores app.vmexe at <base_url><proof_type>/app.vmexe (no VK subdir),
# while the prover's default URL builder appends <proof_type>/<vk>/.
# ASSETS_DIR env override: the mid-run upgrade test (20-upgrade.sh) points this
# at the NEW release's assets dir so the generated prover config carries the
# new VKs.
ASSETS_V2="${ASSETS_DIR:-${REPO_ROOT}/coordinator/build/bin/assets_v2}"
CHUNK_VK=$(jq -r '.chunk_vk' "${ASSETS_V2}/openVmVk.json" 2>/dev/null || echo "")
BATCH_VK=$(jq -r '.batch_vk' "${ASSETS_V2}/openVmVk.json" 2>/dev/null || echo "")
BUNDLE_VK=$(jq -r '.bundle_vk' "${ASSETS_V2}/openVmVk.json" 2>/dev/null || echo "")

# ─── Build prover if needed ──────────────────────────────────────────────────
# PROVER_BIN env override: the mid-run upgrade test points this at the binary
# built from another checkout (Phase 1 = develop/production worktree).
PROVER_BIN="${PROVER_BIN:-${REPO_ROOT}/target/release/prover}"

if $DOCKER; then
    docker image inspect "$PROVER_IMAGE" >/dev/null 2>&1 || {
        log_error "missing image $PROVER_IMAGE — build it first: docker build -f build/dockerfiles/prover.Dockerfile -t $PROVER_IMAGE ."
        exit 1
    }
    log_info "  prover image: $PROVER_IMAGE"
elif [[ ! -f "$PROVER_BIN" ]]; then
    log_info "Building prover (GPU)..."
    cd "${REPO_ROOT}/zkvm-prover"
    make prover
    log_ok "  Built: $PROVER_BIN"
fi

# ─── Launch provers ──────────────────────────────────────────────────────────
IFS=',' read -ra GPU_ARRAY <<< "$GPUS"

for i in "${!GPU_ARRAY[@]}"; do
    gpu_id="${GPU_ARRAY[$i]}"
    prover_name="${PROVER_NAME}-${gpu_id}"
    work_dir="${SCRIPT_DIR}/../.work/prover-${gpu_id}"
    mkdir -p "$work_dir"
    # Canonicalize (resolve the lib/.. component): docker bind-mount targets are
    # path-cleaned by dockerd, but the prover opens the config by the literal
    # path, which fails inside the container if it still contains "..".
    work_dir="$(cd "$work_dir" && pwd)"
    config_file="${work_dir}/prover.json"
    log_file="${work_dir}/prover.log"

    # Generate per-GPU config
    cat > "$config_file" <<EOF
{
  "sdk_config": {
    "prover_name_prefix": "${prover_name}",
    "keys_dir": "${work_dir}",
    "coordinator": {
      "base_url": "http://localhost:8390",
      "retry_count": 10,
      "retry_wait_time_sec": 10,
      "connection_timeout_sec": 1800
    },
    "prover": {
      "supported_proof_types": [1, 2, 3],
      "circuit_version": "${CIRCUIT_VERSION}"
    },
    "health_listener_addr": "127.0.0.1:$((10080 + gpu_id))",
    "db_path": "${work_dir}/db"
  },
  "circuits": {
    "galileoV2": {
      "base_url": "${S3_URL}",
      "workspace_path": "${work_dir}/galileo",
      "asset_detours": {
        "${CHUNK_VK}": "${S3_URL}chunk/",
        "${BATCH_VK}": "${S3_URL}batch/",
        "${BUNDLE_VK}": "${S3_URL}bundle/"
      },
      "child_circuit_vks": {
        "1": "${CHUNK_VK}",
        "2": "${BATCH_VK}"
      },
      "debug_mode": false
    }
  }
}
EOF

    # Kill existing prover on this GPU. The prover name lives inside the
    # config file, not the cmdline, so match on the config path instead;
    # prefer the pidfile when it is valid. Skipping this leaves the old
    # process holding the LevelDB lock — the new one then dies on startup
    # and the pidfile ends up pointing at a corpse.
    if [[ -f "${work_dir}/prover.pid" ]] && kill -0 "$(cat "${work_dir}/prover.pid")" 2>/dev/null; then
        kill "$(cat "${work_dir}/prover.pid")" 2>/dev/null || true
        sleep 2
    fi
    pkill -f "prover-${gpu_id}/prover.json" 2>/dev/null || true
    sleep 1

    log_info "Starting prover on GPU $gpu_id..."

    export RUST_MIN_STACK=16777216
    if $DOCKER; then
        # Same-path mounts + host network keep the generated config (host
        # absolute paths, localhost coordinator URL) valid inside the
        # container. --user + recorded host PID keep the shared stop
        # scripts (pidfile kill / pkill on the config path) working.
        docker rm -f "shadow-prover-${gpu_id}" >/dev/null 2>&1 || true
        docker run -d --name "shadow-prover-${gpu_id}" --network host \
            --gpus "device=${gpu_id}" \
            --user "$(id -u):$(id -g)" -e HOME=/home/prover \
            -e RUST_MIN_STACK=16777216 \
            -v "${work_dir}:${work_dir}" \
            -v "${HOME}/.openvm/params:/home/prover/.openvm/params:ro" \
            "$PROVER_IMAGE" --config "$config_file" >/dev/null
        sleep 2
        pid=$(docker inspect -f '{{.State.Pid}}' "shadow-prover-${gpu_id}")
        if [[ -z "$pid" || "$pid" = "0" ]]; then
            docker logs "shadow-prover-${gpu_id}" 2>&1 | tail -20 || true
            log_error "  prover container shadow-prover-${gpu_id} failed to start"
            exit 1
        fi
        echo "$pid" > "${work_dir}/prover.pid"
        log_ok "  Prover $gpu_id started in docker (container shadow-prover-${gpu_id}, host PID $pid, health :$((10080 + gpu_id)); logs: docker logs shadow-prover-${gpu_id})"
    else
        CUDA_VISIBLE_DEVICES="$gpu_id" nohup "$PROVER_BIN" \
            --config "$config_file" \
            >> "$log_file" 2>&1 &

        pid=$!
        echo "$pid" > "${work_dir}/prover.pid"
        log_ok "  Prover $gpu_id started (PID $pid, health :$((10080 + gpu_id)))"
    fi
done

log_ok "All provers launched"
