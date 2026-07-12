#!/usr/bin/env bash
# Start prover(s) for shadow testing
# Usage: ./04-prover-up.sh --config mainnet --bundle-range 17297:17301 [--gpus 0,1]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/anvil-utils.sh"

CONFIG="${CONFIG:-mainnet}"
GPUS="${GPUS:-0,1}"
DOCKER=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)      CONFIG="$2"; shift 2 ;;
        --gpus)        GPUS="$2"; shift 2 ;;
        --docker)      DOCKER=true; shift ;;
        -h|--help)     sed -n '2,5p' "$0"; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

CONFIG_FILE="${SCRIPT_DIR}/../configs/${CONFIG}.json"
if [[ ! -f "$CONFIG_FILE" ]]; then
    log_error "Config not found: $CONFIG_FILE"
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
ASSETS_V2="${REPO_ROOT}/coordinator/build/bin/assets_v2"
CHUNK_VK=$(jq -r '.chunk_vk' "${ASSETS_V2}/openVmVk.json" 2>/dev/null || echo "")
BATCH_VK=$(jq -r '.batch_vk' "${ASSETS_V2}/openVmVk.json" 2>/dev/null || echo "")
BUNDLE_VK=$(jq -r '.bundle_vk' "${ASSETS_V2}/openVmVk.json" 2>/dev/null || echo "")

# ─── Build prover if needed ──────────────────────────────────────────────────
PROVER_BIN="${REPO_ROOT}/target/release/prover"

if [[ ! -f "$PROVER_BIN" ]]; then
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
    config_file="${work_dir}/prover.json"
    log_file="${work_dir}/prover.log"

    mkdir -p "$work_dir"

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

    # Kill existing prover on this GPU
    pkill -f "prover.*${prover_name}" 2>/dev/null || true

    log_info "Starting prover on GPU $gpu_id..."

    export RUST_MIN_STACK=16777216
    CUDA_VISIBLE_DEVICES="$gpu_id" nohup "$PROVER_BIN" \
        --config "$config_file" \
        >> "$log_file" 2>&1 &

    pid=$!
    echo "$pid" > "${work_dir}/prover.pid"
    log_ok "  Prover $gpu_id started (PID $pid, health :$((10080 + gpu_id)))"
done

log_ok "All provers launched"
