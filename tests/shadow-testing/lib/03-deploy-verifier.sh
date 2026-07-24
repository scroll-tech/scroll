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
CONFIG_FILE="${PROJECT_ROOT}/follow/configs/mainnet.json"
ASSETS_DIR="${PROJECT_ROOT}/../../coordinator/build/bin/assets_v2"
DB_BUNDLE_INDEX="17302"

deploy_plonk=true
extract_digests=true
deploy_wrapper=true
register=true
genuine_register=false
START_BATCH_OVERRIDE=""

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
        --start-batch)
            START_BATCH_OVERRIDE="$2"; shift 2 ;;
        --genuine-register)
            genuine_register=true; shift ;;
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
fetched from S3) and register it on Anvil.

Options:
  --config <path>         Config file (default: follow/configs/mainnet.json)
  --assets-dir <path>     Path to coordinator assets_v2/ (default: ../../coordinator/build/bin/assets_v2)
  --bundle-index <idx>    Unused legacy option (kept for compatibility)
  --start-batch <N>       Register the new verifier at startBatchIndex=N instead
                          of the computed max(lastFinalized+1, existing start)
  --genuine-register      Register ONLY via the real updateVerifier() contract
                          call (production upgrade path; the previous verifier
                          stays in legacyVerifiers). Default is to also force
                          the latestVerifier storage slot, which bypasses MVRV
                          routing and is wrong for mid-run upgrade tests.
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

# Resolve to absolute path before we cd into scroll-contracts for deployment.
CONFIG_FILE="$(cd "$(dirname "$CONFIG_FILE")" && pwd)/$(basename "$CONFIG_FILE")"

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
EXISTING_START=$(cast call "$MVRV" "latestVerifier(uint256)(uint64,address)" 10 --rpc-url "$ANVIL_RPC" 2>/dev/null | grep -oP '^\d+' | head -1 || echo "0")
MIN_START=$((LAST_FINALIZED + 1))
if [[ "$EXISTING_START" -gt "$MIN_START" ]]; then
    START_BATCH="$EXISTING_START"
else
    START_BATCH="$MIN_START"
fi
if [[ -n "$START_BATCH_OVERRIDE" ]]; then
    START_BATCH="$START_BATCH_OVERRIDE"
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
        S3_BASE_URL=$(jq -r '.prover.s3_base_url // empty' "$CONFIG_FILE")
        if [[ -n "$S3_BASE_URL" ]]; then
            log_info "  Downloading plonk verifier binary from S3..."
            mkdir -p "$ASSETS_DIR"
            curl -fsSL "${S3_BASE_URL}verifier/verifier.bin" -o "$VERIFIER_BIN" || true
        fi
    fi

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
# 2. Fetch digests from S3 release
# ---------------------------------------------------------------------------
DIGEST1=""
DIGEST2=""
if $extract_digests; then
    S3_BASE_URL=$(jq -r '.prover.s3_base_url // empty' "$CONFIG_FILE")
    if [[ -z "$S3_BASE_URL" ]]; then
        log_error "Missing prover.s3_base_url in config; cannot download digest files"
        exit 1
    fi

    log_info "Fetching digests from S3: ${S3_BASE_URL}bundle/digest_*.hex"

    # CAUTION: digest files are usable as-is ONLY for v0.9.0+ (canonical form).
    # v0.8.0 S3 digests are Montgomery-encoded and must be converted first —
    # see tests/shadow-testing/docs/bundle-digest-encoding.md.
    DIGEST1_HEX=$(curl -fsSL "${S3_BASE_URL}bundle/digest_1.hex" 2>/dev/null | tr -d '[:space:]')
    DIGEST2_HEX=$(curl -fsSL "${S3_BASE_URL}bundle/digest_2.hex" 2>/dev/null | tr -d '[:space:]')

    if [[ -z "$DIGEST1_HEX" || -z "$DIGEST2_HEX" ]]; then
        log_error "Failed to download digest files from S3"
        exit 1
    fi

    DIGEST1="0x${DIGEST1_HEX}"
    DIGEST2="0x${DIGEST2_HEX}"

    log_info "Fetched digest1: $DIGEST1"
    log_info "Fetched digest2: $DIGEST2"
else
    log_error "--skip-digests not supported; digests must be fetched from S3"
    exit 1
fi

# ---------------------------------------------------------------------------
# 3. Deploy ZkEvmVerifierPostFeynman wrapper
# ---------------------------------------------------------------------------
WRAPPER_ADDR=""
PROTOCOL_VERSION=$(jq -r '.reset.codec_version // 10' "$CONFIG_FILE")
if $deploy_wrapper; then
    log_info "Deploying ZkEvmVerifierPostFeynman ..."
    log_info "  plonkVerifier:   $PLONK_VERIFIER"
    log_info "  digest1:         $DIGEST1"
    log_info "  digest2:         $DIGEST2"
    log_info "  protocolVersion: $PROTOCOL_VERSION"

    cd "${PROJECT_ROOT}/../../scroll-contracts"

    # The wrapper was moved from the scroll-contracts submodule to the shadow-testing
    # directory, so deploy from the absolute path instead of a relative one.
    WRAPPER_CONTRACT_PATH="${PROJECT_ROOT}/contracts/ZkEvmVerifierPostFeynman.sol"
    if [[ ! -f "$WRAPPER_CONTRACT_PATH" ]]; then
        log_error "Wrapper contract not found: $WRAPPER_CONTRACT_PATH"
        exit 1
    fi

    WRAPPER_OUTPUT=$(forge create --broadcast --evm-version cancun --rpc-url "$ANVIL_RPC" \
        --from "$OWNER" --unlocked \
        "$WRAPPER_CONTRACT_PATH:ZkEvmVerifierPostFeynman" \
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
    ONCHAIN_PROTOCOL_VERSION=$(cast call "$WRAPPER_ADDR" "protocolVersion()(uint256)" --rpc-url "$ANVIL_RPC")

    log_info "On-chain verification:"
    log_info "  plonkVerifier:   $ONCHAIN_PLONK"
    log_info "  digest1:         $ONCHAIN_DIGEST1"
    log_info "  digest2:         $ONCHAIN_DIGEST2"
    log_info "  protocolVersion: $ONCHAIN_PROTOCOL_VERSION"
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

    if $genuine_register; then
        # Production upgrade path: a single updateVerifier() call. The contract
        # moves the current latest verifier into legacyVerifiers, so batches
        # below START_BATCH still route to the OLD verifier — this is exactly
        # what a mid-run upgrade test needs. No storage-slot forcing.
        UPDATE_CALLDATA=$(cast calldata "updateVerifier(uint256,uint64,address)" 10 "$START_BATCH" "$WRAPPER_ADDR")
        UPDATE_TX=$(cast rpc eth_sendTransaction \
            "{\"from\":\"$OWNER\",\"to\":\"$MVRV\",\"data\":\"$UPDATE_CALLDATA\",\"gas\":\"0x4c4b40\"}" \
            --rpc-url "$ANVIL_RPC" 2>/dev/null | tr -d '"')
        if [[ -z "$UPDATE_TX" || "$UPDATE_TX" != 0x* ]]; then
            log_error "updateVerifier eth_sendTransaction failed (impersonated owner $OWNER)"
            exit 1
        fi
        # Wait for the tx to seal, then check it did not revert.
        for _ in $(seq 1 20); do
            cast receipt "$UPDATE_TX" --rpc-url "$ANVIL_RPC" >/dev/null 2>&1 && break
            sleep 0.5
        done
        TX_STATUS=$(cast receipt "$UPDATE_TX" status --rpc-url "$ANVIL_RPC" 2>/dev/null || echo "")
        if [[ "$TX_STATUS" != "0x1" && "$TX_STATUS" != "1" ]]; then
            log_error "updateVerifier reverted (tx $UPDATE_TX, status '$TX_STATUS')"
            exit 1
        fi
        log_info "updateVerifier tx: $UPDATE_TX"
    else
        # The contract enforces startBatchIndex >= existing latest.startBatchIndex.
        # On a shadow fork the production verifier may already be registered at a
        # later batch (e.g. 517767), so a normal updateVerifier at 517761 would be
        # rejected or ignored. We therefore force the storage slot for
        # latestVerifier[10] to point to our wrapper at the desired start batch.
        # This is acceptable for a local shadow fork because we only care about the
        # target batch range.
        #
        # NOTE: this bypasses MVRV legacy routing. Mid-run upgrade tests must use
        # --genuine-register instead, otherwise the old verifier is lost.

        # latestVerifier is state slot 2 (slot 0 = Ownable owner, slot 1 = legacyVerifiers).
        LATEST_VERIFIER_SLOT=$(cast index uint256 10 2 2>/dev/null)
        START_BATCH_HEX=$(printf '%016x' "$START_BATCH")
        WRAPPER_NO_0X=${WRAPPER_ADDR#0x}
        NEW_SLOT_VALUE="0x00000000${WRAPPER_NO_0X}${START_BATCH_HEX}"

        cast rpc anvil_setStorageAt "$MVRV" "$LATEST_VERIFIER_SLOT" "$NEW_SLOT_VALUE" --rpc-url "$ANVIL_RPC" >/dev/null 2>&1

        # Also push the previous production verifier into legacyVerifiers so that
        # getVerifier still behaves reasonably for older batches. This is optional
        # but keeps the MVRV state closer to reality.
        cast send "$MVRV" \
            "updateVerifier(uint256,uint64,address)" \
            10 "$START_BATCH" "$WRAPPER_ADDR" \
            --from "$OWNER" --rpc-url "$ANVIL_RPC" --unlocked >/dev/null 2>&1 || true
    fi

    # Verify registration for the target batch range
    for BATCH_IDX in $START_BATCH $((START_BATCH + 1)) $((START_BATCH + 2)) $((START_BATCH + 3)) $((START_BATCH + 4)); do
        REGISTERED=$(cast call "$MVRV" "getVerifier(uint256,uint256)(address)" 10 "$BATCH_IDX" --rpc-url "$ANVIL_RPC")
        log_info "getVerifier(10, $BATCH_IDX) = $REGISTERED"
        if [[ "${REGISTERED,,}" != "${WRAPPER_ADDR,,}" ]]; then
            log_error "Registration verification failed for batch $BATCH_IDX!"
            exit 1
        fi
    done

    log_info "Registration verified ✅"
fi

# ---------------------------------------------------------------------------
# 5. Update config file
# ---------------------------------------------------------------------------
log_info "Updating config file: $CONFIG_FILE"
tmp=$(mktemp)
jq --arg addr "$WRAPPER_ADDR" '.contracts.deployed_verifier = $addr' "$CONFIG_FILE" > "$tmp" && mv "$tmp" "$CONFIG_FILE"
log_info "Config updated with deployed_verifier = $WRAPPER_ADDR"

# Also record the deployment in .work/verifier.env so monitor-catchup.py
# (verifier drift check) and re-fork.sh can find the wrapper without parsing
# the config file.
mkdir -p "${PROJECT_ROOT}/.work"
cat > "${PROJECT_ROOT}/.work/verifier.env" <<EOF
WRAPPER_ADDR=${WRAPPER_ADDR}
PLONK_VERIFIER=${PLONK_VERIFIER}
PROTOCOL_VERSION=${PROTOCOL_VERSION}
DEPLOYED_AT=$(date -u +%Y-%m-%dT%H:%M:%S%z)
EOF
log_info "Wrote ${PROJECT_ROOT}/.work/verifier.env"

log_info "Done! 🎉"
log_info ""
log_info "Summary:"
log_info "  Plonk Verifier:  $PLONK_VERIFIER"
log_info "  Wrapper:         $WRAPPER_ADDR"
log_info "  Digest1:         $DIGEST1"
log_info "  Digest2:         $DIGEST2"
