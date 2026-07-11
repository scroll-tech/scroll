# Shadow Coordinator + Prover Testing Guide

This guide documents how to set up a **shadow coordinator** + **local prover** environment for testing proof generation without interfering with production. This approach is significantly simpler than a full shadow fork — we use a local coordinator with imported production task data and a local prover that fetches tasks from it.

## Architecture

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  Production RDS  │     │  Shadow DB       │     │  Shadow          │
│  (read-only via  │────▶│  (local :5433)   │────▶│  Coordinator     │
│   port-forward)  │     │                  │     │  (localhost:8390)│
└──────────────────┘     └──────────────────┘     └────────┬─────────┘
                                                           │
                                                           │ assigns tasks
                                                           ▼
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  L2 RPC          │     │  Local Prover    │     │  Verifier Assets │
│  (mainnet-rpc.   │◀────│  (GPU/CPU)       │     │  (/tmp/shadow-   │
│   scroll.io)     │     │                  │     │   verifier-assets)│
└──────────────────┘     └──────────────────┘     └──────────────────┘
```

## Prerequisites

### Hardware
- GPU with CUDA support (tested on RTX 3090)
- ~50GB disk space for Docker images + verifier assets + circuit downloads
- 16GB+ RAM

### Software
- Docker + docker-compose
- PostgreSQL client (`psql`)
- Rust toolchain (for local prover binary)
- `kubectl` or SSH access to IDC for port-forwarding to production RDS

### Network
- Access to IDC machine with port-forward to mainnet RDS (e.g., `idc-us-1-19`)
- Internet access for L2 RPC and S3 circuit downloads

## Quick Start

If you just want to get running, use the provided script:

```bash
# 1. Set up shadow PostgreSQL
cd tests/shadow-testing
./setup.sh --postgres

# 2. Import production task data (requires RDS port-forward)
./import-production-data.sh

# 3. Start shadow coordinator
./setup.sh --coordinator

# 4. Start prover (in another terminal)
./setup.sh --prover
```

## Step-by-Step Setup

### Step 1: Set up IDC Port-Forward to Production RDS

On the IDC machine (e.g., `idc-us-1-19`), ensure the port-forward is active:

```bash
# Mainnet RDS should be accessible on localhost:15432
# Credentials are loaded from .env (see .env.example)
psql -h localhost -p 15432 -U "$PROD_DB_USER" -d rollup -c "SELECT version();"
```

If not already set up, configure SSH tunnel or kubectl port-forward from your workstation.

### Step 2: Start Local PostgreSQL (Shadow DB)

```bash
docker run -d \
  --name shadow-coordinator-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD="${SHADOW_DB_PASSWORD}" \
  -e POSTGRES_DB=shadow_rollup \
  -p 5433:5432 \
  -v shadow-coordinator-postgres-data:/var/lib/postgresql/data \
  postgres:15

# Wait for DB to be ready
sleep 5
docker exec shadow-coordinator-postgres pg_isready -U postgres
```

### Step 3: Download Verifier Assets

The coordinator needs verifier assets for each supported fork:

```bash
VERIFIER_DIR="/tmp/shadow-verifier-assets"
mkdir -p "$VERIFIER_DIR"

# feynman (OpenVM 0.5.6)
mkdir -p "$VERIFIER_DIR/openvm-0.5.6"
# Download or copy verifier assets for feynman

# galileo (v0.7.1)
mkdir -p "$VERIFIER_DIR/openvm-v0.7.1"
# Download or copy verifier assets for galileo

# galileoV2 (v0.9.0)
mkdir -p "$VERIFIER_DIR/openvm-v0.9.0"
# Download or copy verifier assets for galileoV2 from .../scroll-zkvm/releases/v0.9.0/verifier/
```

> ⚠️ **Important**: v0.9.0 assets are under `scroll-zkvm/releases/v0.9.0/`. Earlier v0.8.0 assets used `scroll-zkvm/v0.8.0/` (no `/releases/`). Using the wrong prefix causes HTTP 403 errors.

### Step 4: Initialize Shadow DB Schema

Use the coordinator's built-in migration or apply schema manually. The coordinator container will auto-migrate on first start.

### Step 5: Import Production Task Data

Export the latest N batches + their chunks + bundles from production RDS and import into shadow DB:

```bash
# Edit these variables as needed
# Credentials loaded from .env (see tests/shadow-testing/.env.example)
PROD_DB="postgresql://${PROD_DB_USER}:${PROD_DB_PASSWORD}@${PROD_DB_HOST}:${PROD_DB_PORT}/${PROD_DB_NAME}"
SHADOW_DB="postgresql://${SHADOW_DB_USER}:${SHADOW_DB_PASSWORD}@${SHADOW_DB_HOST}:${SHADOW_DB_PORT}/${SHADOW_DB_NAME}"
BATCH_LIMIT=50

# Export batches
psql "$PROD_DB" -c "
  COPY (
    SELECT * FROM batch
    ORDER BY index DESC
    LIMIT $BATCH_LIMIT
  ) TO STDOUT WITH CSV HEADER;
" > /tmp/batches.csv

# Export chunks in those batches
psql "$PROD_DB" -c "
  COPY (
    SELECT c.* FROM chunk c
    JOIN batch b ON b.start_chunk_index <= c.index AND c.index <= b.end_chunk_index
    WHERE b.index IN (SELECT index FROM batch ORDER BY index DESC LIMIT $BATCH_LIMIT)
    ORDER BY c.index
  ) TO STDOUT WITH CSV HEADER;
" > /tmp/chunks.csv

# Export bundles (all or limited)
psql "$PROD_DB" -c "
  COPY (
    SELECT * FROM bundle
    ORDER BY index DESC
    LIMIT 20000
  ) TO STDOUT WITH CSV HEADER;
" > /tmp/bundles.csv

# Import into shadow DB (truncate first)
psql "$SHADOW_DB" -c "TRUNCATE batch, chunk, bundle CASCADE;"

# Use \copy for local import
psql "$SHADOW_DB" -c "\\copy batch FROM '/tmp/batches.csv' WITH CSV HEADER;"
psql "$SHADOW_DB" -c "\\copy chunk FROM '/tmp/chunks.csv' WITH CSV HEADER;"
psql "$SHADOW_DB" -c "\\copy bundle FROM '/tmp/bundles.csv' WITH CSV HEADER;"

# Reset proving status to unassigned (1)
psql "$SHADOW_DB" -c "UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0;"
psql "$SHADOW_DB" -c "UPDATE batch SET proving_status = 1, total_attempts = 0, active_attempts = 0, chunk_proofs_status = 0;"
psql "$SHADOW_DB" -c "UPDATE bundle SET proving_status = 1, total_attempts = 0, active_attempts = 0;"
```

### Step 6: Populate l2_block Table

The coordinator needs `l2_block` records to format chunk tasks (for block hashes and hardfork name resolution).

Use the provided Python script or fetch blocks via L2 RPC:

```bash
python3 tests/shadow-testing/scripts/fetch-l2-blocks.py \
  --rpc https://mainnet-rpc.scroll.io \
  --db "postgresql://$SHADOW_DB_USER:$SHADOW_DB_PASSWORD@$SHADOW_DB_HOST:$SHADOW_DB_PORT/$SHADOW_DB_NAME" \
  --start-block 26000000 \
  --end-block 27000000
```

After inserting blocks, link them to chunks:

```bash
psql "$SHADOW_DB" -c "
  UPDATE l2_block lb
  SET chunk_hash = c.hash
  FROM chunk c
  WHERE lb.number >= c.start_block_number
    AND lb.number <= c.end_block_number;
"
```

### Step 7: Start Shadow Coordinator

Use Docker (recommended) or run locally:

```bash
# Via Docker
docker run -d \
  --name shadow-coordinator-api-test \
  --network host \
  -v /tmp/shadow-coordinator-config.json:/app/conf/config.json \
  -v /tmp/shadow-verifier-assets:/verifier \
  zhuoatscroll/coordinator-api:v4.7.13-openvm16

# Wait for startup (takes 2-3 min for OpenVM keygen)
docker logs -f shadow-coordinator-api-test | grep -m1 "Start coordinator api successfully"
```

### Step 8: Start Prover

Build or use prebuilt binary:

```bash
# Build locally
cd /path/to/scroll-repo
cargo build --release -p prover-bin

# Or use Docker image
docker run -d \
  --name shadow-prover \
  --network host \
  --gpus all \
  -v /tmp/prover-local.json:/app/config.json \
  -v ~/.openvm/params:/root/.openvm/params:ro \
  zhuoatscroll/prover:v4.7.13-openvm16

# Or run binary directly
./target/release/prover --config /tmp/prover-local.json
```

> ℹ️ **Note**: Prover will download circuit assets from S3 on first run (several GB). Subsequent runs use cached assets in `.work/galileo/`.

## Monitoring

### Check coordinator health
```bash
curl -s http://localhost:8390/ | head
```

### Check prover health
```bash
curl -s http://localhost:10080/health
```

### Watch coordinator logs
```bash
docker logs -f shadow-coordinator-api-test --tail 100
```

### Watch prover logs
```bash
# If running via docker
docker logs -f shadow-prover --tail 100

# If running binary directly, logs go to stdout
```

### Check DB task status
```bash
psql "$SHADOW_DB" -c "
  SELECT proving_status, COUNT(*) FROM chunk GROUP BY proving_status;
"
```

Proving status values:
- `1` = Unassigned
- `2` = Assigned
- `3` = Proving
- `4` = Proven (success)
- `5` = Failed

### Verify bundle readiness before finalization

After all batch proofs are done (status 4), verify the bundle-level status:

```bash
psql "$SHADOW_DB" -c "
  SELECT index, batch_proofs_status, finalization_status
  FROM bundle WHERE index = 13470;
"
```

Bundle `batch_proofs_status` values:
- `1` = Pending — waiting for coordinator cron to scan and transition
- `2` = Ready — all batch proofs verified, bundle can be finalized

The coordinator runs `checkBundleAllBatchReady()` every 10s to transition bundles from `1` to `2`. If this hasn't happened yet (e.g., cron is lagging), manually update:

```sql
UPDATE bundle SET batch_proofs_status = 2 WHERE index = 13470;
```

> **Important**: The relayer will not call `finalizeBundleWithProof` until `batch_proofs_status >= 2`, even if all individual batch proofs are already status 4.

## Troubleshooting

### Coordinator says "Start coordinator api successfully" but prover gets no tasks
- Verify `l2_block` table has records for the chunk's block range
- Check `proving_status = 1` on chunks
- Check `codec_version != 5` (chunks with codec_version = 5 are skipped)
- Ensure chunk's `end_block_number <= coordinator's block height`

### "mismatched post-state root" or codec errors
- Verify you're using blocks after the hardfork. For GalileoV2 (codec V10), use blocks ≥ 33,750,000 on mainnet.
- Ensure `SCROLL_FORK_NAME` and verifier assets match the block's fork.

### "Failed to execute witness" or "Method not found"
- The L2 RPC must support `debug_executionWitness` and `debug_dbGet`.
- `https://mainnet-rpc.scroll.io` supports these; `https://rpc.scroll.io` does not.

### "Failed to get l1 messages in block" (-32601)
- Your RPC does not support `scroll_getL1MessagesInBlock`. This is non-fatal if the block contains no L1 messages.
- If L1 messages exist, you need an RPC that supports this method.

### S3 403 errors when downloading circuit assets
- v0.9.0 assets: `https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/v0.9.0/`
- v0.8.0 assets: `https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/v0.8.0/`
- v0.7.1 and earlier: `https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/v0.7.1/`
- Verify with `curl -sI <url>` before running.

### "bind: address already in use" (port 8390)
- Kill old coordinator: `pkill -f coordinator_api` or `docker rm -f shadow-coordinator-api-test`

### Port conflicts with local PostgreSQL
- If you have system PostgreSQL on 5432, use 5433 for shadow DB (already configured).
- Ensure all configs use the correct port.

### Multi-GPU prover cache conflicts
When running multiple prover instances on the same machine, the shared `.work/galileo` cache directory can cause `File exists (os error 17)` conflicts if two provers write the same temp file simultaneously.

**Mitigation**: Ensure each prover has its own work directory, or symlink `.work/galileo` to a shared read-only cache while giving each instance a distinct write directory. Example launch script:
```bash
for i in 0 1 2 3; do
  mkdir -p /tmp/prover-gpu${i}/work
  ln -s /shared/cache/galileo /tmp/prover-gpu${i}/work/galileo
  CUDA_VISIBLE_DEVICES=$i ./prover --config /tmp/prover-gpu${i}/config.json &
done
```

### Bundle proving never starts
If coordinator is actively assigning chunk/batch tasks but never assigns bundle tasks, the most likely cause is **orphan bundles** — bundle records whose corresponding batch data no longer exists in the shadow DB.

**Diagnosis**:
```sql
-- Count bundles with no linked batches
SELECT COUNT(*) FROM bundle b
WHERE NOT EXISTS (
  SELECT 1 FROM batch bat
  WHERE bat.index BETWEEN b.start_batch_index AND b.end_batch_index
);
```

**Root cause**: The bundle table often retains historical records from production (e.g., batch 308516+) while the batch table only holds recently imported batches (e.g., 517760+). Coordinator's `GetUnassignedBundle` picks the lowest-index bundle with `batch_proofs_status = 2`, finds it has no batches, and fails silently in a loop.

**Fix**:
```sql
UPDATE bundle
SET batch_proofs_status = 1
WHERE index NOT IN (
    SELECT DISTINCT b.index
    FROM bundle b
    JOIN batch bat ON bat.index BETWEEN b.start_batch_index AND b.end_batch_index
);
```

### DB data inconsistency after import
If imported chunks have `proving_status = 2` (assigned) but `proof = NULL`, coordinator may incorrectly set `batch.chunk_proofs_status = 2` and then fail when formatting batch tasks.

**Fix**:
```sql
UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0
WHERE proving_status = 2 AND proof IS NULL;

UPDATE batch SET chunk_proofs_status = 0
WHERE chunk_proofs_status != 0
  AND EXISTS (
    SELECT 1 FROM chunk c
    WHERE c.batch_hash = batch.hash AND c.proving_status != 4
  );
```

## Configuration Reference

### Shadow Coordinator Config

See `configs/shadow-coordinator-config.json` in this directory.

Key fields:
- `db.dsn`: Points to shadow PostgreSQL
- `l2.l2geth.endpoint`: L2 RPC with `debug_executionWitness` support
- `prover_manager.verifier.verifiers`: List of verifier asset paths and fork names

### Prover Config

See `configs/prover-local.json` in this directory.

Key fields:
- `sdk_config.coordinator.base_url`: Shadow coordinator API (`http://localhost:8390`)
- `circuits.galileoV2.base_url`: S3 path for circuit assets (`.../scroll-zkvm/releases/v0.9.0/` for v0.9.0)
- `sdk_config.prover.supported_proof_types`: `[1, 2, 3]` for chunk, batch, bundle

## Rollup Relayer Dry-Run Mode

For testing the **rollup-relayer's transaction construction logic** (e.g., `finalizeBundle` calldata) without spending real gas or modifying chain state, the sender module supports a **dry-run mode**.

When `"dry_run": true` is set in the sender config:
- Transactions are **simulated** via `eth_call` instead of being broadcast
- `pending_transaction` table is **not** populated (avoids DB pollution)
- Nonce is still incremented to simulate real behavior
- If the `eth_call` fails (e.g., contract revert), the error is propagated just like a real send failure

### Usage

1. Build the rollup-relayer binary:
```bash
cd rollup && go build -o rollup_relayer ./cmd/rollup_relayer/app
```

2. Configure `dry_run: true` in the sender config (see `tests/shadow-testing/configs/rollup-relayer-dryrun.json`)

3. Start the relayer:
```bash
./rollup_relayer --config /path/to/rollup-relayer-dryrun.json
```

### What Dry-Run Verifies

| Aspect | Verified? | Notes |
|--------|-----------|-------|
| Calldata encoding (ABI pack) | ✅ | `constructFinalizeBundlePayloadCodecV7` etc. |
| Gas estimation | ✅ | Full `EstimateGas` + `CreateAccessList` path |
| Contract revert | ✅ | `eth_call` returns revert reason |
| Signature / nonce | ⚠️ | Nonce incremented but tx not broadcast |
| Pending tx lifecycle | ❌ | Skipped to avoid DB pollution |
| Receipt confirmation | ❌ | No real tx = no receipt |

For **full end-to-end** validation (including signature + receipt), use **Anvil** with `evm_snapshot`/`evm_revert` instead.

### Anvil + Mock ScrollChain Setup (Recommended for Dry-Run)

For the most realistic dry-run testing, deploy a minimal mock ScrollChain contract on a local Anvil node:

```bash
# 1. Start Anvil forked from mainnet (or standalone)
anvil --fork-url https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY --fork-block-number 33878313

# 2. Deploy mock contract (minimal Solidity with no-op commitBatches / finalizeBundle)
cat > MockScrollChain.sol << 'EOF'
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;
contract MockScrollChain {
    mapping(address => bool) public isProver;
    address public owner;
    constructor() { owner = msg.sender; }
    function addProver(address _prover) external {
        require(msg.sender == owner, "Not owner");
        isProver[_prover] = true;
    }
    function commitBatches(uint8 version, bytes32 parentBatchHash, bytes32 batchHash) external {}
    function finalizeBundlePostEuclidV2NoProof(bytes calldata, uint256, bytes32, bytes32) external {}
    function finalizeBundlePostEuclidV2(bytes calldata, uint256, bytes32, bytes32, bytes calldata) external {}
}
EOF

# Compile and deploy
solc --bin MockScrollChain.sol -o /tmp/mock
BYTECODE=$(cat /tmp/mock/MockScrollChain.bin)
cast send --rpc-url http://localhost:18545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  --create "0x$BYTECODE"
# → contractAddress: 0x1fA02b2d6A771842690194Cf62D91bdd92BfE28d

# 3. Fund sender accounts and add prover
COMMIT_ADDR="0x1e32ABcfE6db15c1570709E3fC02725335f50A47"
FINALIZE_ADDR="0x33e0F539E31B35170FAaA062af703b76a8282bf7"
cast rpc anvil_setBalance "$COMMIT_ADDR" "0x3635c9adc5dea00000" --rpc-url http://localhost:18545
cast rpc anvil_setBalance "$FINALIZE_ADDR" "0x3635c9adc5dea00000" --rpc-url http://localhost:18545
cast send <MOCK_ADDR> "addProver(address)" "$FINALIZE_ADDR" --rpc-url http://localhost:18545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

**Key sender config changes**:
```json
{
  "sender_config": {
    "endpoint": "http://localhost:18545",
    "dry_run": true
  }
}
```

**Dry-run gas estimation skip**: Anvil may fail `EstimateGas` on blob transactions or missing functions. A small patch to `rollup/internal/controller/sender/estimategas.go` skips gas estimation in dry-run mode:
```go
func (s *Sender) estimateGasLimit(...) (uint64, *types.AccessList, error) {
    if s.config.DryRun {
        return 10000000, nil, nil  // skip estimation
    }
    // ... original logic
}
```

### What We Verified in Practice

| Transaction | Status | Notes |
|-------------|--------|-------|
| `commitBatches` | ✅ `eth_call` succeeded | Selector `0x9bbaa2ba` via mock `commitBatches(uint8,bytes32,bytes32)` |
| `finalizeBundlePostEuclidV2NoProof` | ✅ `eth_call` succeeded | Selector `0xbd6f916b` via mock no-op |
| `finalizeBundlePostEuclidV2` (with proof) | ✅ `eth_call` succeeded | Bundle 17301 with valid `OpenVMBundleProof` |

### ⚠️ Critical Discovery: Anvil Must Fork Ethereum Mainnet, NOT Scroll Mainnet

When querying `0xa13BAF47339d63B743e7Da8741db5456DAc1E556` on **Scroll L2** (`scroll-mainnet.g.alchemy.com`), the contract appears to have no ScrollChain functions and an empty implementation slot. This led to confusion — the address seemed to be a ProxyAdmin rather than the ScrollChain proxy.

**The root cause**: We were querying the **wrong chain**. The ScrollChain proxy `0xa13B...` is deployed on **Ethereum L1**, not Scroll L2. When queried on Ethereum mainnet:

- **Implementation**: `0x0a20703878e68e587c59204cc0ea86098b8c3ba7` (ScrollChain logic)
- **Admin**: `0xEB803eb3F501998126bf37bB823646Ed3D59d072` (ProxyAdmin)
- **Functions verified**: `lastFinalizedBatchIndex()`, `committedBatches(uint256)`, `isSequencer(address)`, `isProver(address)`, `commitBatches(uint8,bytes32,bytes32)`, `finalizeBundlePostEuclidV2(bytes,uint256,bytes32,bytes32,bytes)`

### Real ScrollChain Proxy Dry-Run Testing

For testing against the **actual deployed ScrollChain contract** on an Anvil fork:

```bash
# 1. Start Anvil forked from ETHEREUM mainnet (NOT Scroll mainnet)
anvil --fork-url https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY \
  --fork-block-number 25206000 \
  --port 18545 \
  --no-rate-limit \
  --block-time 5

# 2. Run takeover script (impersonate owner, add sequencer/prover)
# See scroll-devnets/charts/shadow-fork/rollup-relayer/scripts/takeover-l1-contracts.sh
# Key addresses:
#   L1_SCROLL_CHAIN_PROXY_ADDR=0xa13BAF47339d63B743e7Da8741db5456DAc1E556
#   L1_SCROLL_OWNER_ADDR=0x798576400F7D662961BA15C6b3F3d813447a26a6
#   FORKED_L1_SCROLL_OWNER_ADDR=0x909D2900A1Ec2B518EAFe11811Da0c1Fc8729a73
#   FORKED_L1_SCROLL_OWNER_PRIVATE_KEY=0x93d9b2e68479131dfa877a77cef8a286986940ab2de677a4790d17267462dd5e

# 3. Set balances for relayer senders
COMMIT_ADDR="0x1e32ABcfE6db15c1570709E3fC02725335f50A47"
FINALIZE_ADDR="0x33e0F539E31B35170FAaA062af703b76a8282bf7"
cast rpc anvil_setBalance "$COMMIT_ADDR" "0x21e19e0c9bab2400000" --rpc-url http://localhost:18545
cast rpc anvil_setBalance "$FINALIZE_ADDR" "0x21e19e0c9bab2400000" --rpc-url http://localhost:18545

# 4. Configure relayer to use REAL proxy address
# In config: "rollup_contract_address": "0xa13BAF47339d63B743e7Da8741db5456DAc1E556"
```

**Important**: If blob base fee is extremely high on the forked block (causing `Insufficient funds`), mine empty blocks to reduce `excessBlobGas`:
```bash
cast rpc anvil_mine 400 --rpc-url http://localhost:18545
```

### Dry-Run Results with Real ScrollChain Proxy

| Transaction | Status | Notes |
|-------------|--------|-------|
| `commitBatches` | ⚠️ `eth_call` reached contract | Reverted with `ErrorIncorrectBatchHash()` — shadow DB batch data is ahead of fork block state |
| `finalizeBundlePostEuclidV2` | ✅ **Succeeded** | Bundle 17330 (batch 517809) finalized successfully with real mainnet proof. See "End-to-End finalizeBundlePostEuclidV2 Dry-Run Success" below. |

**Why `commitBatches` reverts**: The shadow DB contains batches 518565+ but the Anvil fork block (25206000) only has batches committed up to ~517816. The parent batch hash in the calldata doesn't match what the contract expects, triggering `ErrorIncorrectBatchHash()`.

This is **expected and actually confirms the pipeline works** — the relayer is successfully constructing and sending calldata to the real ScrollChain implementation, and the contract's validation logic is executing correctly.

For `finalizeBundlePostEuclidV2`, the batch was already committed on mainnet at the fork block, so no `commitBatches` call is needed — we only need the proof and verifier to match.

---

## Real Verifier Deployment

### Option 1: Copy the Mainnet Verifier (Fastest)

For quick testing, copy the exact mainnet verifier contract code to Anvil using `anvil_setCode`:

```bash
# Copy mainnet ZkEvmVerifierPostFeynman wrapper (0x0dE1...)
MAINNET_VERIFIER="0x0dE180164Dc571522457101F5c47B2eaB36d0A82"
CODE=$(cast code $MAINNET_VERIFIER --rpc-url https://ethereum-rpc.publicnode.com)
cast rpc anvil_setCode $MAINNET_VERIFIER $CODE --rpc-url http://localhost:18545

# Copy its Plonk verifier (0x749f...)
PLONK="0x749fc77a1a131632a8b88e8703e489557660c75e"
PLONK_CODE=$(cast code $PLONK --rpc-url https://ethereum-rpc.publicnode.com)
cast rpc anvil_setCode $PLONK $PLONK_CODE --rpc-url http://localhost:18545
```

This preserves the exact immutables (plonkVerifier address, digests, protocolVersion) from mainnet and works as long as your proofs use the same digests as mainnet.

### Option 2: Deploy a Fresh Verifier Using S3 Digests

When testing a new guest / circuit version (e.g., v0.9.0), deploy a fresh `ZkEvmVerifierPostFeynman` with digests taken from the release S3 bucket:

```bash
BASE_URL="https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/v0.9.0"
DIGEST1=$(curl -fsSL "${BASE_URL}/bundle/digest_1.hex" | tr -d '[:space:]')
DIGEST2=$(curl -fsSL "${BASE_URL}/bundle/digest_2.hex" | tr -d '[:space:]')

forge create --broadcast --evm-version cancun --rpc-url http://localhost:18545 \
  --from "$OWNER" --unlocked \
  src/libraries/verifier/ZkEvmVerifierPostFeynman.sol:ZkEvmVerifierPostFeynman \
  --constructor-args "$PLONK_VERIFIER" "0x$DIGEST1" "0x$DIGEST2" 10
```

For v0.9.0 release assets are under `scroll-zkvm/releases/v0.9.0/`:

| File | S3 Path |
|------|---------|
| Chunk/batch/bundle circuits | `.../releases/v0.9.0/{chunk,batch,bundle}/<vk>/` |
| Verifier assets | `.../releases/v0.9.0/verifier/{openVmVk.json,verifier.bin,root_verifier_vk}` |
| Bundle digests | `.../releases/v0.9.0/bundle/{digest_1.hex,digest_2.hex}` |

> ✅ **Use the S3 digest files directly.** For guest v0.9.0, `digest_1.hex` / `digest_2.hex` are published in the canonical form expected by the Plonk verifier. You no longer need to extract digests from a proof's `instances` array.

### Verifying Digests Match Your Proofs

If you want to double-check, compare the canonical digests from S3 against the values stored in the deployed wrapper:

```bash
cast call "$WRAPPER" "verifierDigest1()(bytes32)" --rpc-url "$ANVIL_RPC"
cast call "$WRAPPER" "verifierDigest2()(bytes32)" --rpc-url "$ANVIL_RPC"
```

These must match the digests published at `.../releases/v0.9.0/bundle/{digest_1.hex,digest_2.hex}`.

### Register the Verifier

```bash
MVRV="0x4cea3e866e7c57fd75cb0ca3e9f5f1151d4ead3f"
OWNER="0x909d2900a1ec2b518eafe11811da0c1fc8729a73"
ANVIL_VERIFIER="0x0dE180164Dc571522457101F5c47B2eaB36d0A82"

# Impersonate owner and register
cast rpc anvil_impersonateAccount $OWNER --rpc-url http://localhost:18545
cast send $MVRV \
  "updateVerifier(uint256,uint64,address)" \
  10 0 $ANVIL_VERIFIER \
  --from $OWNER --rpc-url http://localhost:18545 --unlocked
```

> **Note**: `latestVerifier[10]` returns a struct; use `getVerifier(10, batchIndex)` to confirm routing.

### Verify MVRV Routing

After registering a new verifier, **always verify that MVRV routes target batches to the correct verifier** before starting finalization tests. This is especially critical when testing new prover digests on batch ranges that may already be mapped to a legacy verifier.

```bash
# Check which verifier MVRV returns for each batch in your target range
for idx in 128069 128070 128071; do
  echo -n "Batch $idx → "
  cast call $MVRV "getVerifier(uint256,uint256)(address)" 10 $idx --rpc-url http://localhost:18545
done
```

If any batch returns the **wrong verifier** (e.g., an old production verifier whose digests don't match your proofs), update the MVRV mapping:

```bash
# Route batches ≥ START_BATCH to the new verifier
START_BATCH=128069
NEW_VERIFIER="0x16110D4e0CBE54530cE46D1aB2b22574BeEEa105"

cast rpc anvil_impersonateAccount $OWNER --rpc-url http://localhost:18545
cast send $MVRV \
  "updateVerifier(uint256,uint64,address)" \
  10 $START_BATCH $NEW_VERIFIER \
  --from $OWNER --rpc-url http://localhost:18545 --unlocked
```

> **Critical**: If MVRV routes to the wrong verifier, finalization will revert with `VerificationFailed(0x439cc0cd)` even though your deployed verifier and proof digests are correct.

### Critical Discovery: Anvil `eth_call` vs `anvil_setStorageAt`

**Refined conclusion** (updated after further testing):

- `anvil_setStorageAt` on **mapping slots** (e.g., `committedBatches[batchIndex]`) is visible to `eth_getStorageAt` but is **cached and ignored** by `eth_call` / `eth_sendTransaction` during contract execution. This is an Anvil bug.
- `anvil_setStorageAt` on **direct variable slots** (e.g., `miscData` at slot 161, `nextUnfinalizedQueueIndex` at slot 104) **does work** and is visible to `eth_call`.

**Implications**:
- You **can** override simple state variables like `lastFinalizedBatchIndex`, `nextUnfinalizedQueueIndex`, etc.
- You **cannot** override mapping entries like `committedBatches[517809]` or `finalizedStateRoots[517808]`.
- For mappings, either fork at a block where the desired state already exists, or use a mock contract.

### Deployed Contract Addresses (Anvil Fork)

| Contract | Address | Notes |
|----------|---------|-------|
| ScrollChain Proxy | `0xa13BAF47339d63B743e7Da8741db5456DAc1E556` | Forked from mainnet |
| MultipleVersionRollupVerifier | `0x4CEA3E866e7c57fD75CB0CA3E9F5f1151D4Ead3F` | Forked from mainnet |
| **ZkEvmVerifierPostFeynman (v10)** | `0x0dE180164Dc571522457101F5c47B2eaB36d0A82` | **Copied from mainnet** ✅ |
| Plonk Verifier (v10) | `0x749fc77a1a131632a8b88e8703e489557660c75e` | Copied from mainnet |
| ZkEvmVerifierPostFeynman (wrong) | `0xc3230A4C89a5Ce0455414215e533de4D8849b3f8` | Deployed with S3 digests — **do not use** |

---

## End-to-End finalizeBundlePostEuclidV2 Dry-Run Success

We successfully executed `finalizeBundlePostEuclidV2` end-to-end on Anvil using **real mainnet proof data** from shadow DB bundle 17330.

### Bundle 17330 Parameters

| Field | Value |
|-------|-------|
| Bundle index | 17330 |
| Batch index | 517809 |
| Codec version | 10 (GalileoV2) |
| Num batches | 1 |
| `postStateRoot` | `0x28ff638e237ad6a0f2eebaab84f254dd4fca8a16297413c29fcd70f8b1b3fd85` |
| `withdrawRoot` | `0xe88d24e9153438c91f94c32026cb49730212f32ac4652367b07c71f96ce063d9` |
| `batchHash` | `0xeadeee9af865c6d13df6b66a45b3f3f161e6211aeb7d86e075a645f0e6a58f9e` |
| `prevStateRoot` | `0x4d21a5ca662bffc2d650a4d24a445617c3eb7159a28b13548ec5421a3ba08ee7` |
| `prevBatchHash` | `0xd6d7d027ef32d393a4aff7b04c1577bcb1f7fdc44834797e48f7e01581615a58` |
| `totalL1MessagesPoppedOverall` | 998288 |
| `msgQueueHash` | `0x5b08e5befde15d3acbf1a3e0e99622a6ac3fa62049cdfa62ba984ab700000000` |
| Mainnet finalize tx | `0x753f8f9ca01d4e67f710c6dab8ce0b17a17a7ad46a9d7480d92657803a36ca24` |

### Public Input Verification

The 204-byte public input is constructed as:

```
chain_id(8) || msg_queue_hash(32) || num_batches(4) || prev_state_root(32) || prev_batch_hash(32) || post_state_root(32) || batch_hash(32) || withdraw_root(32)
```

The `ZkEvmVerifierPostFeynman` contract prepends `protocolVersion = 10` (32 bytes) and computes:

```solidity
publicInputHash = keccak256(abi.encodePacked(protocolVersion, publicInput))
```

- `protocolVersion` = 10 (GalileoV2)
- `publicInput` = 204 bytes (standard EuclidV2 format)
- Actual input to keccak256 = **236 bytes** (32-byte version prefix + 204-byte public input)

Computed hash: `0xcd4421bad526bd108d9ae8c2af3d46ea1a986207f0b8c1af781b601c1ae50e5a`

This **exactly matches** `bundle_pi_hash` from the proof metadata.

#### Bundle Proof Format

The `aggrProof` passed to `finalizeBundlePostEuclidV2` is:

```
bundleProof = instances[:384] + proof_bytes  = 1760 bytes total
├── 384 bytes  = accumulator (12 Fr elements)
└── 1376 bytes = Plonk proof
```

The `ZkEvmVerifierPostFeynman.verify()` function inserts `digest1`, `digest2`, and `publicInputHash` expansion into the calldata before forwarding to the Plonk verifier.

### Pre-Execution Setup Required

Because the Anvil fork block (25213457) is **after** the real finalization block (25198501), several state variables had already advanced past the values needed for the dry-run. We applied the following fixes:

#### 1. Add Authorized Prover

The prover authorization was lost after `anvil_reset`. Re-add:

```bash
SCROLL_CHAIN="0xa13BAF47339d63B743e7Da8741db5456DAc1E556"
OWNER="0x798576400F7D662961BA15C6b3F3d813447a26a6"
PROVER="0xc48DfbcdC4ef4cdACFf94eE7385020b7a7CE195f"

cast rpc anvil_setBalance $OWNER 0x56bc75e2d63100000 --rpc-url http://localhost:18545
cast send $SCROLL_CHAIN "addProver(address)" $PROVER \
  --from $OWNER --rpc-url http://localhost:18545 --unlocked
```

> ⚠️ **Anvil Default Account Is Not an EOA**: Anvil's default test account `0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266` has contract code (`0xef0100...`) in fork mode. `ScrollChain.addProver()` checks `_account.code.length == 0` and will revert with `ErrorAccountIsNotEOA`. Always use a freshly generated EOA (e.g., from `cast wallet new`) as the prover address.
>
> If the real owner account is an EOA with delegation code (e.g., ERC-7702), you may need to temporarily swap the proxy owner via `anvil_setStorageAt` on slot 51, call `addProver`, then restore the original owner.

#### 2. Override `lastFinalizedBatchIndex`

Set `miscData` (slot 161) so `lastFinalizedBatchIndex = 517808`:

```bash
cast rpc anvil_setStorageAt $SCROLL_CHAIN 0xa1 \
  0x0000000000000000000000016a1bb977000000000007e6b0000000000007e6d3 \
  --rpc-url http://localhost:18545
```

> Layout: `lastCommitted(8) | lastFinalized(8) | lastFinalizeTimestamp(4) | flags(1) | reserved(7)`

#### 3. Override `L1MessageQueueV2.nextUnfinalizedQueueIndex`

Set slot 104 to `0` (the finalize call will update it to 998288):

```bash
MQV2="0x56971da63A3C0205184FEF096E9ddFc7A8C2D18a"
cast rpc anvil_setStorageAt $MQV2 0x68 0x0 --rpc-url http://localhost:18545
```

#### 4. Copy Mainnet Verifier

See "Real Verifier Deployment" above for the `anvil_setCode` commands to copy the mainnet verifier wrapper and its Plonk verifier.

### Execution

```bash
# Extract proof from mainnet finalize transaction
python3 << 'PYEOF'
import subprocess, json
result = subprocess.run([
    'cast', 'tx', '0x753f8f9ca01d4e67f710c6dab8ce0b17a17a7ad46a9d7480d92657803a36ca24',
    '--json', '--rpc-url', 'https://ethereum-rpc.publicnode.com'
], capture_output=True, text=True)
tx = json.loads(result.stdout)
input_hex = tx['input']
data = bytes.fromhex(input_hex[2:])
# ... decode batchHeader, totalL1MessagesPoppedOverall, postStateRoot, withdrawRoot, aggrProof
PYEOF

# Send transaction
SCROLL_CHAIN="0xa13BAF47339d63B743e7Da8741db5456DAc1E556"
PROVER="0xc48DfbcdC4ef4cdACFf94eE7385020b7a7CE195f"

cast rpc anvil_setBalance $PROVER 0x56bc75e2d63100000 --rpc-url http://localhost:18545
cast send $SCROLL_CHAIN --from $PROVER $(cat /tmp/finalize_calldata.hex) \
  --rpc-url http://localhost:18545 --unlocked
```

### Result

- **Transaction Hash**: `0x0000ba738dbcc27e89db8e545532cdc125a9d50c42683032d92ed30203ea8d65`
- **Status**: Success ✅
- **Gas Used**: 425,719
- **Block**: 25213522

### Post-Execution State

| Variable | Value |
|----------|-------|
| `lastFinalizedBatchIndex` | 517809 |
| `finalizedStateRoots[517809]` | `0x28ff638e237ad6a0f2eebaab84f254dd4fca8a16297413c29fcd70f8b1b3fd85` |
| `L1MessageQueueV2.nextUnfinalizedQueueIndex` | 998288 |

### Key Takeaways

1. **Always deploy `ZkEvmVerifierPostFeynman` with S3 digests** — For guest v0.9.0 the canonical digests are published at `.../releases/v0.9.0/bundle/digest_*.hex`. Use them directly; do not attempt to copy the mainnet verifier wrapper (`anvil_setCode` preserves the original immutables and will fail verification).
2. **`anvil_setStorageAt` works for direct variables** but not for mapping entries. Use it for `miscData`, `nextUnfinalizedQueueIndex`, etc.
3. **Fork block matters** — If the fork block is after the real finalization, you must manually reset `lastFinalizedBatchIndex` and `nextUnfinalizedQueueIndex`.
4. **Public input hash must match exactly** — Any discrepancy in `msg_queue_hash`, `chain_id`, `num_batches`, or roots will cause `VerificationFailed`.
5. **Anvil default account is not an EOA in fork mode** — Use a freshly generated EOA for `addProver`; `0xf39F...` has contract code and will fail the EOA check.
6. **Reset `rollup_status` before relayer finalize** — The shadow DB retains mainnet rollup state (`RollupFinalized` = 5). The relayer's `GetFirstPendingBundle` only queries `rollup_status = RollupPending` (1). You must reset both `bundle` and `batch` tables before the relayer will pick up bundles for finalization.

## Multi-Bundle Relayer Finalize Test (5 Bundles)

This test demonstrates running the actual `rollup_relayer` binary against an Anvil mainnet fork to finalize **5 consecutive bundles** (17297–17301, batches 517761–517765) using shadow proofs.

### Prerequisites

- Anvil fork running with `lastFinalizedBatchIndex` reset to `517760`
- Shadow proofs generated for all 5 bundles (`proving_status = 4`)
- Verifier `0xb1F2C5c1ea2885278a1070350d12d3D8824265B0` registered as `latestVerifier[10]`
- Prover/finalize EOA `0x410E...` authorized on `ScrollChain`

### Step 1: Reset DB Rollup Status

The shadow DB retains mainnet rollup state. Before the relayer can pick up bundles, reset their status:

```sql
UPDATE bundle SET rollup_status = 1 WHERE index BETWEEN 17297 AND 17301;
UPDATE batch SET rollup_status = 1 WHERE index BETWEEN 517761 AND 517765;
```

(`1` = `RollupPending`; without this, `GetFirstPendingBundle` returns nothing.)

### Step 2: Build and Configure Relayer

```bash
cd rollup
go build -o /tmp/rollup_relayer ./cmd/rollup_relayer
```

Create `/tmp/rollup-relayer-anvil.json`:

```json
{
  "l2_config": {
    "l2_geth": { "endpoint": "https://mainnet-galileo.scroll.io/l2" },
    "relayer_config": {
      "sender_config": {
        "endpoint": "http://localhost:18545",
        "check_balance": false,
        "dry_run": false
      },
      "commit_sender_signer_config": {
        "private_key": "0xac09..."
      },
      "finalize_sender_signer_config": {
        "private_key": "0x01f1..."
      },
      "rollup_contract_address": "0xa13BAF47339d63B743e7Da8741db5456DAc1E556",
      "chain_monitor": { "enabled": false },
      "gas_oracle": { "enabled": false },
      "batch_committer": {
        "enable_test_env_bypass_features": true
      },
      "validium_mode": false
    }
  },
  "db_config": {
    "dsn": "postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup"
  }
}
```

**Important**: `commit_sender` and `finalize_sender` must be **different addresses**. The relayer enforces this at startup.

### Step 3: Launch Relayer

```bash
/tmp/rollup_relayer \
  --config /tmp/rollup-relayer-anvil.json \
  --genesis /home/scroll/zzhang/scroll/tests/prover-e2e/mainnet-galileoV2/genesis.json \
  --min-codec-version 7 \
  --verbosity 3 \
  2>&1 | tee /tmp/relayer.log
```

The relayer starts all modules (L2 watcher, proposers, batch committer, bundle finalizer). The batch committer will fail with `ErrorCallerIsNotSequencer` (expected — the commit sender is not a sequencer), but the **bundle finalizer runs independently every 15 seconds** and will pick up the pending bundles.

### Step 4: Monitor Finalization

Watch `/tmp/relayer.log` for:

```
{"msg":"Start to roll up zk proof","index":17297,...}
{"msg":"finalizeBundle in layer1","index":17297,"tx hash":"0x6d62...","with proof":"true"}
```

### Results

| Bundle | Batch | Transaction Hash | Status | Gas Used |
|--------|-------|------------------|--------|----------|
| 17297 | 517761 | `0x6d6264...cdaa725` | ✅ Success | 439,987 |
| 17298 | 517762 | `0x071268...1136516` | ✅ Success | 407,455 |
| 17299 | 517763 | `0x8f8894...6cabd5` | ✅ Success | 407,479 |
| 17300 | 517764 | `0xa87721...302cd3` | ✅ Success | 407,419 |
| 17301 | 517765 | `0x41ee42...c9cf89` | ✅ Success | 401,404 |

**Final `lastFinalizedBatchIndex`**: `517765` (was `517760`)

All 5 bundles finalized consecutively without manual intervention. Each bundle proof was verified on-chain by the `ZkEvmVerifierPostFeynman` contract deployed at `0xb1F2C5c1ea2885278a1070350d12d3D8824265B0`.

### Key Differences from CLI Approach

| Aspect | CLI (`cast send`) | Relayer |
|--------|-------------------|---------|
| Calldata construction | Manual Python script | Relayer reads from DB + constructs automatically |
| Sender management | Single EOA | Separate commit/finalize senders |
| Batch status tracking | None | Updates `bundle` and `batch` `rollup_status` in DB |
| Error handling | Manual retry | Built-in retry and status polling |
| Multi-bundle support | One at a time | Processes all pending bundles automatically |

## Known Limitations

1. **L1 messages**: If chunks contain L1 messages, the prover needs `scroll_getL1MessagesInBlock` RPC support. Most public RPCs don't expose this. Workaround: select chunks/blocks with no L1 messages, or use an internal RPC. In non-validium mode, the prover does not call this RPC at all.

2. **Full batch proving**: Batch tasks require `chunk_proofs_status = 2` (all chunks proven). For quick chunk-only testing, you don't need to prove full batches.

3. **Coordinator startup time**: First startup performs OpenVM keygen (~2-3 min). Be patient.

4. **Circuit download**: First prover run downloads ~5-10GB of circuit assets. Ensure good internet.

5. **Bundle vs batch count mismatch**: The shadow DB's `bundle` table may contain 10,000+ historical records while `batch` only holds ~500 recent ones. This is expected when importing production data — the bundle table retains full history but batches are truncated. **Crucially**, orphan bundles (those with no matching batches) must have `batch_proofs_status = 1` or coordinator will deadlock trying to prove them. See "Bundle proving never starts" in Troubleshooting.

6. **`finalizeBundlePostEuclidV2` and multi-batch bundles**: The contract computes `numBatches = batchIndex - lastFinalizedBatchIndex`. The proof's `num_batches` must exactly match this value. Single-batch bundles (e.g., bundle 17330 = batch 517809) are the easiest to test because `numBatches = 1`. Multi-batch bundles also work as long as `lastFinalizedBatchIndex` is set so that `batchIndex - lastFinalizedBatchIndex` equals the proof's `num_batches`.

7. **Local E2E proofs cannot be used on mainnet fork**: Local E2E proofs are generated against a different chain state (genesis batch, different state roots, different message queue). Even if you deploy matching verifier digests, the public input (state roots, batch hashes, message queue hash) will not match the forked mainnet contract state, causing `VerificationFailed`.

## Automated DB Replication from Mainnet RDS

The `~/.pgpass` file on this machine contains valid credentials for the mainnet RDS read-only replica:

```bash
# Verify access
cast psql -h localhost -p 15432 -U mainnet_infra_team_read_only -d mainnet_rollup -c "SELECT COUNT(*) FROM batch;"
# → 517,830 batches
```

For automated DB sync, see `scroll-devnets/charts/shadow-fork/rollup-relayer/scripts/copy-db.sh` which uses `postgres-tunnel` to stream data from mainnet RDS to local shadow DB via `COPY ... TO STDOUT | COPY ... FROM STDIN`.

## Common DB Fixes

After importing production data or running for extended periods, these SQL fixes resolve common coordinator deadlocks:

### 1. Reset proving status after import
```sql
UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0;
UPDATE batch SET proving_status = 1, total_attempts = 0, active_attempts = 0, chunk_proofs_status = 0;
UPDATE bundle SET proving_status = 1, total_attempts = 0, active_attempts = 0;
```

### 2. Mark orphan bundles (no linked batches)
```sql
UPDATE bundle
SET batch_proofs_status = 1
WHERE index NOT IN (
    SELECT DISTINCT b.index
    FROM bundle b
    JOIN batch bat ON bat.index BETWEEN b.start_batch_index AND b.end_batch_index
);
```

### 3. Fix stale assigned chunks without proofs
```sql
UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0
WHERE proving_status = 2 AND proof IS NULL;

UPDATE batch SET chunk_proofs_status = 0
WHERE chunk_proofs_status != 0
  AND EXISTS (
    SELECT 1 FROM chunk c
    WHERE c.batch_hash = batch.hash AND c.proving_status != 4
  );
```

## Scripts Reference

| Script | Purpose |
|--------|---------|
| `setup.sh` | One-command setup for PostgreSQL, coordinator, or prover |
| `import-production-data.sh` | Export from production RDS and import to shadow DB |
| `fetch-l2-blocks.py` | Fetch block headers from L2 RPC and populate `l2_block` table |
