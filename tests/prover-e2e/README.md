# Prover E2E: Local End-to-End Proving Test

Sets up a local environment (PostgreSQL + coordinator + prover) to run the full chunk → batch → bundle proving pipeline with real block data.

## Available Scenarios

| Scenario | Fork | Network | RPC Endpoint | Accessibility |
|----------|------|---------|-------------|---------------|
| `mainnet-galileoV2` | galileoV2 (codec V10) | Mainnet | `https://mainnet-rpc.scroll.io` | **Public** — recommended |
| `mainnet-galileo` | galileo (codec V9) | Mainnet | `https://mainnet-rpc.scroll.io` | Public — pre-fork blocks only |
| `sepolia-galileoV2` | galileoV2 | Sepolia | `l2geth-rpc-0.sepolia.scroll.tech:8545` | Scroll internal |
| `sepolia-galileo` | galileo | Sepolia | Scroll internal | Scroll internal |
| `cloak-galileoV2` | galileoV2 | Cloak | Scroll internal | Scroll internal |

For most testing, **use `mainnet-galileoV2`** — it works from any machine with internet access.

### Fork vs Block Range

Blocks must be from after the fork height for the configured codec version to work.

| Fork | Codec Version | Mainnet Fork Block (approx.) | Usable Mainnet Blocks |
|------|---------------|------------------------------|----------------------|
| galileo | V9 | Genesis | Any (e.g., 26,653,680+) |
| galileoV2 | V10 | ~33,700,000 | ≥ 33,750,000 |

Using pre-fork blocks with a newer codec produces `mismatched post-state root` errors.

## Quick Start (mainnet-galileoV2)

```bash
cd tests/prover-e2e

# 1. Choose scenario
ln -snf mainnet-galileoV2 conf

# 2. Check environment (port, tools, docker)
make check-env   # if available

# 3. Setup DB + import blocks
make all

# 4. Setup coordinator with downloaded verifier assets
make coordinator_setup

# 5. Launch coordinator
cd ../../coordinator/build/bin
# Edit conf/config.json if needed
LD_LIBRARY_PATH="$(pwd)/../internal/logic/libzkp/lib:$LD_LIBRARY_PATH" ./coordinator_api

# 6. In another terminal, run prover
cd zkvm-prover
# Copy config.template.json → config.json, set coordinator base_url to http://localhost:8390
make test_e2e_run       # CPU
# or
make test_e2e_run_gpu   # GPU (set CUDA_VISIBLE_DEVICES first)
```

## Typical Proving Times (GPU, RTX 3090, blocks 33750000–33750005)

| Task Type | Count | Per-Task Time | Notes |
|-----------|-------|---------------|-------|
| Chunk | 4 | 20–40s | STARK proof |
| Batch | 2 | 60–80s | Aggregates chunk proofs |
| Bundle | 1 | ~18 min | Includes Halo2 SNARK (~4 min) + EVM proof (~2.5 min) |

The bundle SNARK phase is single-threaded (Halo2). Ensure `chunk_collection_time_sec` in the coordinator config is set high enough (≥ 3600) for GPU proving.

## DB Port Configuration

The DB port is controlled by a single variable `DB_PORT` in `.env`. Default is `5432`.

To change the port:
1. Edit `DB_PORT=5432` → `DB_PORT=<your_port>` in `.env`
2. Run `make gen-config` to regenerate config files
3. Run `make setup_db` to restart PostgreSQL on the new port

All config files (docker-compose.yml, goose env, e2e_tool config, coordinator config) derive their port from this single source.

## Test Data

| Scenario | Block Range | Chunks | Batches | Bundles |
|----------|-------------|--------|---------|---------|
| mainnet-galileoV2 | 33,750,000–33,750,005 | 4 | 2 | 1 |
| mainnet-galileo | 26,653,680–26,653,686 | 4 | 2 | 1 |
| sepolia-galileoV2 | ~17,086,000–17,086,005 | varies | varies | 1 |
| cloak-galileoV2 | varies | varies | varies | 1 |

## Troubleshooting

### Port already in use (5432)
```bash
ss -tlnp | grep <port>           # find what's using the port
# Change DB_PORT in .env to an unused port, then make gen-config && make setup_db
```

### Coordinator won't start — bind: address already in use (8390)
```bash
pkill -f coordinator_api          # kill stale processes
ss -tlnp | grep 8390              # verify port is free
```

### Coordinator panics on startup (InitL2geth)
Check `l2geth.endpoint` in `coordinator/build/bin/conf/config.json`. It must be a reachable RPC URL, not a placeholder.

### DB connection refused
```bash
docker port local_postgres        # verify mapping is correct
docker rm -f local_postgres       # kill stale container
make setup_db                     # recreate
```

> After changing `docker-compose.yml`, old containers can persist with stale port mappings — always `docker rm -f` before `docker compose up`. The E2E container is named `local_postgres`.

### Prover 403 downloading app.vmexe from S3
- Verify `base_url` in prover `config.json` doesn't contain an extra `releases/` segment.
- Test with: `curl -sI "<base_url>chunk/<vk>/app.vmexe"`
- S3 prefixes/digest encodings differ per release (v0.8.0 has no `/releases/`; v0.9.0+ requires `agg_vk.bin` per circuit) — see [`../shadow-testing/docs/CURRENT-STACK.md`](../shadow-testing/docs/CURRENT-STACK.md) and shadow-testing follow-mode Trap 33.

### mismatched post-state root (during coordinator task generation)
Blocks are from before the configured fork. Use a higher block range (≥ 33,750,000 for galileoV2).

### invalid data length for DABatchV7 (73 vs 137 bytes)
`validium_mode` is inconsistent between e2e_tool config and coordinator config. Set both to the same value. For mainnet: `false`.

### solc: Invalid option --evm-version cancun
System solc is too old (< 0.8.24). Download newer version:
```bash
curl -sL https://github.com/ethereum/solidity/releases/download/v0.8.24/solc-static-linux -o /tmp/solc
chmod +x /tmp/solc
PATH="/tmp:$PATH" make ...
```

### goose: command not found
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
# Ensure ~/go/bin is in PATH
```
