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

## Known Limitations

1. **L1 messages**: If chunks contain L1 messages, the prover needs `scroll_getL1MessagesInBlock` RPC support. Most public RPCs don't expose this. Workaround: select chunks/blocks with no L1 messages, or use an internal RPC.

2. **Full batch proving**: Batch tasks require `chunk_proofs_status = 2` (all chunks proven). For quick chunk-only testing, you don't need to prove full batches.

3. **Coordinator startup time**: First startup performs OpenVM keygen (~2-3 min). Be patient.

4. **Circuit download**: First prover run downloads ~5-10GB of circuit assets. Ensure good internet.

## Scripts Reference

| Script | Purpose |
|--------|---------|
| `setup.sh` | One-command setup for PostgreSQL, coordinator, or prover |
| `import-production-data.sh` | Export from production RDS and import to shadow DB |
| `fetch-l2-blocks.py` | Fetch block headers from L2 RPC and populate `l2_block` table |
