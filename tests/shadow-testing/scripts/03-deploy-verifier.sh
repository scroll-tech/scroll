#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
CONFIG_FILE="${PROJECT_ROOT}/configs/mainnet.json"
ASSETS_DIR="${PROJECT_ROOT}/../../coordinator/build/bin/assets_v2"
DB_BUNDLE_INDEX="17302"

deploy_plonk=true
extract_digests=true
deploy_wrapper=true
register=true

# ---------------------------------------------------------------------------
# Parse args
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)
            CONFIG_FILE="$2"; shift 2 ;;
        --assets-dir)
            ASSETS_DIR="$2"; shift 2 ;;
        --bundle-index)
            DB_BUNDLE_INDEX="$2"; shift 2 ;;
        --skip-plonk)
            deploy_plonk=false; shift ;;
        --skip-wrapper)
            deploy_wrapper=false; shift ;;
        --skip-register)
            register=false; shift ;;
        --help|-h)
            cat << 'USAGE'
Usage: 03-deploy-verifier.sh [options]

Deploy a new ZkEvmVerifierPostFeynman (with new plonk verifier + digests
extracted from the DB proof) and register it on Anvil.

Options:
  --config <path>         Config file (default: configs/mainnet.json)
  --assets-dir <path>     Path to coordinator assets_v2/ (default: ../../coordinator/build/bin/assets_v2)
  --bundle-index <idx>    Bundle index to extract digests from (default: 17302)
  --skip-plonk            Skip deploying a new plonk verifier (reuse existing)
  --skip-wrapper          Skip deploying the ZkEvmVerifierPostFeynman wrapper
  --skip-register         Skip registering on MultipleVersionRollupVerifier
  -h, --help              Show this help
USAGE
            exit 0 ;;
        *)
            log_error "Unknown option: $1"
            exit 1 ;;
    esac
done

# ---------------------------------------------------------------------------
# Load config
# ---------------------------------------------------------------------------
if [[ ! -f "$CONFIG_FILE" ]]; then
    log_error "Config file not found: $CONFIG_FILE"
    exit 1
fi

ANVIL_RPC=$(jq -r '.fork.anvil_rpc // empty' "$CONFIG_FILE")
SCROLL_CHAIN=$(jq -r '.contracts.scroll_chain // empty' "$CONFIG_FILE")
MVRV=$(jq -r '.contracts.rollup_verifier // empty' "$CONFIG_FILE")
OWNER=$(jq -r '.contracts.owner // empty' "$CONFIG_FILE")
DB_DSN=$(jq -r '.db.dsn // empty' "$CONFIG_FILE")
LAST_FINALIZED=$(jq -r '.reset.last_finalized_batch_index // empty' "$CONFIG_FILE")

if [[ -z "$ANVIL_RPC" || -z "$MVRV" || -z "$OWNER" || -z "$DB_DSN" ]]; then
    log_error "Missing required fields in config"
    exit 1
fi

# Compute start batch: must be >= lastFinalized + 1 AND >= existing latestVerifier startBatchIndex
EXISTING_START=$(cast call "$MVRV" "latestVerifier(uint256)(uint64,address)" 10 --rpc-url "$ANVIL_RPC" 2>/dev/null | grep -oP '^\d+' || echo "0")
MIN_START=$((LAST_FINALIZED + 1))
if [[ "$EXISTING_START" -gt "$MIN_START" ]]; then
    START_BATCH="$EXISTING_START"
else
    START_BATCH="$MIN_START"
fi

log_info "Anvil RPC:      $ANVIL_RPC"
log_info "MVRV:           $MVRV"
log_info "Owner:          $OWNER"
log_info "Min start:      $MIN_START"
log_info "Existing start: $EXISTING_START"
log_info "Using start:    $START_BATCH"
log_info "Bundle index:   $DB_BUNDLE_INDEX"

# ---------------------------------------------------------------------------
# Pre-flight: unlock owner for all impersonated transactions
# ---------------------------------------------------------------------------
cast rpc anvil_impersonateAccount "$OWNER" --rpc-url "$ANVIL_RPC" >/dev/null 2>&1 || true

# ---------------------------------------------------------------------------
# 1. Deploy new Plonk Verifier from assets_v2/verifier.bin
# ---------------------------------------------------------------------------
PLONK_VERIFIER=""
if $deploy_plonk; then
    VERIFIER_BIN="${ASSETS_DIR}/verifier.bin"
    if [[ ! -f "$VERIFIER_BIN" ]]; then
        log_error "Plonk verifier binary not found: $VERIFIER_BIN"
        log_error "Make sure coordinator assets are downloaded (run coordinator once or download from S3)."
        exit 1
    fi

    PLONK_BYTECODE=$(xxd -p "$VERIFIER_BIN" | tr -d '\n')
    log_info "Deploying plonk verifier from $VERIFIER_BIN ($((${#PLONK_BYTECODE} / 2)) bytes) ..."

    # Anvil allows impersonation of any address with --unlocked
    PLONK_DEPLOY_OUTPUT=$(cast send --rpc-url "$ANVIL_RPC" --chain 1 \
        --from "$OWNER" --unlocked --create "$PLONK_BYTECODE" 2>&1)

    # Extract deployed contract address from output
    PLONK_VERIFIER=$(echo "$PLONK_DEPLOY_OUTPUT" | grep -oP 'contractAddress\s+\K0x[a-fA-F0-9]{40}' || true)

    if [[ -z "$PLONK_VERIFIER" ]]; then
        log_error "Failed to extract plonk verifier address from cast output. Raw output:"
        echo "$PLONK_DEPLOY_OUTPUT"
        exit 1
    fi

    log_info "Plonk verifier deployed at: $PLONK_VERIFIER"
else
    # If skipping plonk deploy, read existing deployed_verifier and extract its plonkVerifier
    EXISTING_WRAPPER=$(jq -r '.contracts.deployed_verifier // empty' "$CONFIG_FILE")
    if [[ -n "$EXISTING_WRAPPER" && "$EXISTING_WRAPPER" != "null" ]]; then
        PLONK_VERIFIER=$(cast call "$EXISTING_WRAPPER" "plonkVerifier()(address)" --rpc-url "$ANVIL_RPC" 2>/dev/null || true)
        log_info "Reusing plonk verifier from existing wrapper: $PLONK_VERIFIER"
    fi
    if [[ -z "$PLONK_VERIFIER" ]]; then
        log_error "Cannot determine plonk verifier address. Either deploy one or provide an existing wrapper."
        exit 1
    fi
fi

# ---------------------------------------------------------------------------
# 2. Extract digests from DB proof instances
# ---------------------------------------------------------------------------
DIGEST1=""
DIGEST2=""
if $extract_digests; then
    log_info "Extracting digests from bundle $DB_BUNDLE_INDEX proof instances ..."

    PROOF_JSON=$(psql "$DB_DSN" -Atq -c "
        SELECT encode(proof, 'escape')
        FROM bundle
        WHERE index = $DB_BUNDLE_INDEX;
    " 2>/dev/null)

    if [[ -z "$PROOF_JSON" ]]; then
        log_error "Bundle $DB_BUNDLE_INDEX not found in DB"
        exit 1
    fi

    # Parse instances base64 and extract digests
    DIGESTS=$(echo "$PROOF_JSON" | python3 -c "
import sys, json, base64
data = sys.stdin.read()
j = json.loads(data)
instances_raw = base64.b64decode(j['proof']['instances'])
# instances: 12 accumulators (384) + digest1 (32) + digest2 (32) + publicInputHash bytes (32*32=1024)
digest1 = '0x' + instances_raw[384:416].hex()
digest2 = '0x' + instances_raw[416:448].hex()
print(digest1)
print(digest2)
")

    DIGEST1=$(echo "$DIGESTS" | sed -n '1p')
    DIGEST2=$(echo "$DIGESTS" | sed -n '2p')

    log_info "Extracted digest1: $DIGEST1"
    log_info "Extracted digest2: $DIGEST2"
else
    log_error "--skip-digests not supported; digests must always be extracted from proof"
    exit 1
fi

# ---------------------------------------------------------------------------
# 3. Deploy ZkEvmVerifierPostFeynman wrapper
# ---------------------------------------------------------------------------
WRAPPER_ADDR=""
if $deploy_wrapper; then
    # IMPORTANT: For new guest proofs (v0.8.0+), the correct wrapper is
    # ZkEvmVerifierPostFeynman, NOT ZkEvmVerifierPostEuclid.
    # PostFeynman computes keccak256(abi.encodePacked(protocolVersion, publicInput))
    # which matches the bundle_pi_hash embedded in the proof instances.
    # protocolVersion = (domain << 6) + stf_version = (0 << 6) + 10 = 10 for Scroll+V10.
    PROTOCOL_VERSION=10
    log_info "Deploying ZkEvmVerifierPostFeynman ..."
    log_info "  plonkVerifier:  $PLONK_VERIFIER"
    log_info "  digest1:        $DIGEST1"
    log_info "  digest2:        $DIGEST2"
    log_info "  protocolVersion: $PROTOCOL_VERSION"

    cd "${PROJECT_ROOT}/../../scroll-contracts"

    WRAPPER_OUTPUT=$(forge create --broadcast --evm-version cancun --rpc-url "$ANVIL_RPC" \
        --from "$OWNER" --unlocked \
        src/libraries/verifier/ZkEvmVerifierPostFeynman.sol:ZkEvmVerifierPostFeynman \
        --constructor-args "$PLONK_VERIFIER" "$DIGEST1" "$DIGEST2" "$PROTOCOL_VERSION" 2>&1)

    WRAPPER_ADDR=$(echo "$WRAPPER_OUTPUT" | grep -oP 'Deployed to:\s+\K0x[a-fA-F0-9]{40}' || true)

    if [[ -z "$WRAPPER_ADDR" ]]; then
        log_error "Failed to extract wrapper address from forge output. Raw output:"
        echo "$WRAPPER_OUTPUT"
        exit 1
    fi

    log_info "ZkEvmVerifierPostFeynman deployed at: $WRAPPER_ADDR"

    # Verify on-chain
    ONCHAIN_DIGEST1=$(cast call "$WRAPPER_ADDR" "verifierDigest1()(bytes32)" --rpc-url "$ANVIL_RPC")
    ONCHAIN_DIGEST2=$(cast call "$WRAPPER_ADDR" "verifierDigest2()(bytes32)" --rpc-url "$ANVIL_RPC")
    ONCHAIN_PLONK=$(cast call "$WRAPPER_ADDR" "plonkVerifier()(address)" --rpc-url "$ANVIL_RPC")

    ONCHAIN_PROTO=$(cast call "$WRAPPER_ADDR" "protocolVersion()(uint256)" --rpc-url "$ANVIL_RPC")
    log_info "On-chain verification:"
    log_info "  plonkVerifier:   $ONCHAIN_PLONK"
    log_info "  digest1:         $ONCHAIN_DIGEST1"
    log_info "  digest2:         $ONCHAIN_DIGEST2"
    log_info "  protocolVersion: $ONCHAIN_PROTO"
else
    WRAPPER_ADDR=$(jq -r '.contracts.deployed_verifier // empty' "$CONFIG_FILE")
    log_info "Reusing existing wrapper: $WRAPPER_ADDR"
fi

# ---------------------------------------------------------------------------
# 4. Register on MultipleVersionRollupVerifier
# ---------------------------------------------------------------------------
if $register; then
    log_info "Registering verifier on MultipleVersionRollupVerifier ..."
    log_info "  version:       10"
    log_info "  startBatch:    $START_BATCH"
    log_info "  verifier:      $WRAPPER_ADDR"

    cast send "$MVRV" \
        "updateVerifier(uint256,uint64,address)" \
        10 "$START_BATCH" "$WRAPPER_ADDR" \
        --from "$OWNER" --rpc-url "$ANVIL_RPC" --unlocked

    # Verify registration
    REGISTERED=$(cast call "$MVRV" "getVerifier(uint256,uint256)(address)" 10 "$START_BATCH" --rpc-url "$ANVIL_RPC")
    log_info "getVerifier(10, $START_BATCH) = $REGISTERED"

    if [[ "${REGISTERED,,}" != "${WRAPPER_ADDR,,}" ]]; then
        log_error "Registration verification failed!"
        exit 1
    fi

    log_info "Registration verified ✅"
fi

# ---------------------------------------------------------------------------
# 5. Update config file
# ---------------------------------------------------------------------------
log_info "Updating config file: $CONFIG_FILE"
tmp=$(mktemp)
jq --arg addr "$WRAPPER_ADDR" '.contracts.deployed_verifier = $addr' "$CONFIG_FILE" > "$tmp" && mv "$tmp" "$CONFIG_FILE"
log_info "Config updated with deployed_verifier = $WRAPPER_ADDR"

log_info "Done! 🎉"
log_info ""
log_info "Summary:"
log_info "  Plonk Verifier:  $PLONK_VERIFIER"
log_info "  Wrapper:         $WRAPPER_ADDR"
log_info "  Digest1:         $DIGEST1"
log_info "  Digest2:         $DIGEST2"
