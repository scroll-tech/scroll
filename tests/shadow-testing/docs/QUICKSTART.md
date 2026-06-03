# Quick Start: Shadow Coordinator + Prover

For full details, see `tests/shadow-testing/docs/README.md`.

## Prerequisites

1. **IDC port-forward active**: Mainnet RDS on `localhost:15432`
2. **Docker installed** with GPU support (for prover)
3. **Verifier assets** at `/tmp/shadow-verifier-assets/` (feynman, galileo, galileoV2)
4. **SRS params** at `~/.openvm/params/` (kzg_bn254_22.srs, kzg_bn254_23.srs, kzg_bn254_24.srs)

## One-Command Setup

```bash
cd tests/shadow-testing

# Step 1: Start PostgreSQL
./scripts/setup.sh --postgres

# Step 2: Import production tasks (requires RDS port-forward)
./scripts/import-production-data.sh

# Step 3: Fetch L2 block headers
python3 ./scripts/fetch-l2-blocks.py \
  --rpc https://mainnet-rpc.scroll.io \
  --db "postgresql://<user>:<password>@localhost:5433/shadow_rollup" \
  --start-block 33750000 --end-block 33770000

# Step 4: Link blocks to chunks
psql "postgresql://<user>:<password>@localhost:5433/shadow_rollup" -c "
  UPDATE l2_block lb SET chunk_hash = c.hash
  FROM chunk c
  WHERE lb.number >= c.start_block_number AND lb.number <= c.end_block_number;
"

# Step 5: Start coordinator (takes 2-3 min)
./scripts/setup.sh --coordinator

# Step 6: Start prover (in another terminal)
./scripts/setup.sh --prover
```

## Monitoring

```bash
# Check everything is running
./scripts/setup.sh --status

# Watch coordinator logs
docker logs -f shadow-coordinator-api-test --tail 100

# Watch prover logs (if using docker)
docker logs -f shadow-prover --tail 100

# Check task assignment
psql "postgresql://<user>:<password>@localhost:5433/shadow_rollup" -c "
  SELECT proving_status, COUNT(*) FROM chunk GROUP BY proving_status;
"
```

## Stop Everything

```bash
./scripts/setup.sh --stop
```

## Key Configuration Files

| File | Purpose |
|------|---------|
| `configs/shadow-coordinator-config.json` | Coordinator config template |
| `configs/prover-local.json` | Prover config template |
| `/tmp/shadow-coordinator-config.json` | Generated coordinator config (with L2 RPC) |
| `/tmp/prover-local.json` | Generated prover config |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PROD_DB` | `postgresql://...localhost:15432/rollup` | Production RDS connection |
| `SHADOW_DB` | `postgresql://...localhost:5433/shadow_rollup` | Shadow DB connection |
| `VERIFIER_DIR` | `/tmp/shadow-verifier-assets` | Verifier asset path |
| `IMAGE_TAG` | `v4.7.13-openvm16` | Docker image tag |
| `L2_RPC` | `https://mainnet-rpc.scroll.io` | L2 RPC endpoint |
