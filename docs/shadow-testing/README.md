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
cd scripts/shadow-testing
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

# galileoV2 (v0.8.0) — NOTE: v0.8.0 does NOT use /releases/ prefix in S3 URLs
mkdir -p "$VERIFIER_DIR/openvm-v0.8.0"
# Download or copy verifier assets for galileoV2
```

> ⚠️ **Important**: v0.8.0 assets use `v0.8.0/` path prefix, NOT `releases/v0.8.0/`. Using the wrong prefix causes HTTP 403 errors.

### Step 4: Initialize Shadow DB Schema

Use the coordinator's built-in migration or apply schema manually. The coordinator container will auto-migrate on first start.

### Step 5: Import Production Task Data

Export the latest N batches + their chunks + bundles from production RDS and import into shadow DB:

```bash
# Edit these variables as needed
# Credentials loaded from .env (see scripts/shadow-testing/.env.example)
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
python3 scripts/shadow-testing/fetch-l2-blocks.py \
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
- `circuits.galileoV2.base_url`: S3 path for circuit assets (no `/releases/` for v0.8.0)
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

2. Configure `dry_run: true` in the sender config (see `scripts/shadow-testing/configs/rollup-relayer-dryrun.json`)

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

## Known Limitations

1. **L1 messages**: If chunks contain L1 messages, the prover needs `scroll_getL1MessagesInBlock` RPC support. Most public RPCs don't expose this. Workaround: select chunks/blocks with no L1 messages, or use an internal RPC. In non-validium mode, the prover does not call this RPC at all.

2. **Full batch proving**: Batch tasks require `chunk_proofs_status = 2` (all chunks proven). For quick chunk-only testing, you don't need to prove full batches.

3. **Coordinator startup time**: First startup performs OpenVM keygen (~2-3 min). Be patient.

4. **Circuit download**: First prover run downloads ~5-10GB of circuit assets. Ensure good internet.

5. **Bundle vs batch count mismatch**: The shadow DB's `bundle` table may contain 10,000+ historical records while `batch` only holds ~500 recent ones. This is expected when importing production data — the bundle table retains full history but batches are truncated. **Crucially**, orphan bundles (those with no matching batches) must have `batch_proofs_status = 1` or coordinator will deadlock trying to prove them. See "Bundle proving never starts" in Troubleshooting.

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
