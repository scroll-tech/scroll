# Re-proving a Single Mainnet Chunk

This guide describes how to re-prove one specific chunk that already exists in the mainnet production database, without running a full shadow fork or replaying an entire chain segment.

Typical use cases:

- A chunk was proven with an old guest / circuit version and the existing proof no longer validates against current verifier assets.
- You want to reproduce a mainnet chunk proof locally for debugging or verification.

## High-level idea

The mainnet DB user available to agents is **read-only**, so a local coordinator cannot update `proving_status` or insert `prover_task` records against mainnet RDS directly. The workaround is:

1. Export the target `chunk` row and its `l2_block` rows from mainnet RDS.
2. Import them into the local **shadow DB** (`localhost:5433/shadow_rollup`), which is writable.
3. Run a temporary coordinator against the shadow DB but with **mainnet L2 RPC** and the production verifier assets.
4. Run the prover in `handle` mode pointing at that coordinator and request exactly the chunk hash.
5. Extract the new proof from the shadow DB, verify it independently, and save it.

## Prerequisites

- `psql` and access to both:
  - Mainnet RDS via the local tunnel (`localhost:15432/mainnet_rollup`)
  - Shadow Postgres (`localhost:5433/shadow_rollup`, writable)
- Built coordinator binary: `coordinator/build/bin/coordinator_api`
- Built prover binary: `target/release/prover`
- Verifier assets matching the current circuit version, e.g. `coordinator/build/bin/assets_v2`
- Cached circuit assets for the prover (the S3 `circuit-release` bucket may return 403 from this environment). Look for an existing `.work/galileo/<vk>/` directory and copy it into the prover workspace.
- Mainnet genesis file, e.g. `tests/prover-e2e/mainnet-galileoV2/genesis.json`

## Step-by-step

### 1. Export the chunk and its blocks from mainnet

Use `enable_seqscan = off` when querying `l2_block`; the table is huge and a plain `BETWEEN` scan can time out even with a partial index.

```bash
EXPORT_DIR=/tmp/chunk-6641451-export
mkdir -p $EXPORT_DIR

# chunk row
PGPASSWORD='<mainnet_pass>' psql -h localhost -p 15432 \
  -U mainnet_infra_team_read_only -d mainnet_rollup -Atq \
  -c "COPY (SELECT * FROM chunk WHERE index = 6641451) TO STDOUT WITH CSV HEADER;" \
  > $EXPORT_DIR/chunk.csv

# l2_block rows for the chunk's block range
PGPASSWORD='<mainnet_pass>' psql -h localhost -p 15432 \
  -U mainnet_infra_team_read_only -d mainnet_rollup -Atq \
  -c "
    SET enable_seqscan = off;
    COPY (
      SELECT * FROM l2_block
      WHERE number BETWEEN 34180693 AND 34180761
        AND deleted_at IS NULL
    ) TO STDOUT WITH CSV HEADER;
  " > $EXPORT_DIR/l2_blocks.csv
```

### 2. Import into the shadow DB

```bash
PGPASSWORD='shadow_pass' psql -h localhost -p 5433 \
  -U postgres -d shadow_rollup \
  -f /dev/stdin <<EOF
\copy chunk FROM '$EXPORT_DIR/chunk.csv' WITH CSV HEADER
\copy l2_block FROM '$EXPORT_DIR/l2_blocks.csv' WITH CSV HEADER
EOF
```

### 3. Reset the chunk to unassigned

```bash
PGPASSWORD='shadow_pass' psql -h localhost -p 5433 \
  -U postgres -d shadow_rollup -Atq -c "
    UPDATE chunk
    SET proving_status = 1, total_attempts = 0, active_attempts = 0, proof = NULL
    WHERE index = 6641451;
  "
```

### 4. Start a temporary mainnet coordinator

Create a config that uses the **shadow DB** for writes but the **mainnet L2 RPC** for task formatting:

```json
{
  "prover_manager": {
    "provers_per_session": 1,
    "session_attempts": 100,
    "external_prover_threshold": 10,
    "chunk_collection_time_sec": 3600,
    "batch_collection_time_sec": 2700,
    "bundle_collection_time_sec": 3600,
    "verifier": {
      "min_prover_version": "v4.5.32",
      "verifiers": [
        {
          "assets_path": "/home/scroll/zzhang/scroll/coordinator/build/bin/assets_v2",
          "fork_name": "galileoV2"
        }
      ]
    }
  },
  "db": {
    "driver_name": "postgres",
    "dsn": "postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup?sslmode=disable",
    "maxOpenNum": 200,
    "maxIdleNum": 20
  },
  "l2": {
    "chain_id": 534352,
    "validium_mode": false,
    "l2geth": {
      "endpoint": "https://l2geth-rpc-proxy.mainnet.aws.scroll.io"
    }
  },
  "auth": {
    "secret": "shadow-coordinator-mainnet-secret-key-2026",
    "challenge_expire_duration_sec": 10,
    "login_expire_duration_sec": 3600
  },
  "sequencer": {
    "decryption_key": ""
  }
}
```

Start it with the mainnet genesis and **no timeout**, otherwise long waits will be killed:

```bash
cd coordinator
./build/bin/coordinator_api \
  --config /tmp/chunk-6641451-export/coordinator-mainnet.json \
  --genesis ../tests/prover-e2e/mainnet-galileoV2/genesis.json
```

Use `nohup`, `screen`, or a background task manager with `disable_timeout=true`.

### 5. Prepare the prover handle set

```bash
cat > $EXPORT_DIR/handle-set.json <<'EOF'
{
  "chunks": [
    "0x90861ddc4e0effb030f59644b03c8bd80e0b2ca8632c8c3500a25029fe4f4081"
  ],
  "batches": [],
  "bundles": []
}
EOF
```

### 6. Configure the prover

Key settings:

- `coordinator.base_url`: `http://localhost:8390`
- `circuits.galileoV2.workspace_path`: directory containing the cached circuit assets
- `circuits.galileoV2.debug_mode`: `true` to skip S3 preflight / download (S3 may 403)
- `sdk_config.db_path`: separate from other prover runs

Example:

```json
{
  "sdk_config": {
    "prover_name_prefix": "mainnet-reprove-chunk-6641451",
    "keys_dir": "/tmp/chunk-6641451-export/prover-keys",
    "coordinator": {
      "base_url": "http://localhost:8390",
      "retry_count": 10,
      "retry_wait_time_sec": 10,
      "connection_timeout_sec": 1800
    },
    "prover": {
      "supported_proof_types": [1],
      "circuit_version": "v0.13.1"
    },
    "health_listener_addr": "127.0.0.1:10080",
    "db_path": "/tmp/chunk-6641451-export/prover-db"
  },
  "circuits": {
    "galileoV2": {
      "base_url": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/v0.9.0/",
      "workspace_path": "/tmp/prover-multi-gpu/gpu0/.work/galileo",
      "debug_mode": true
    }
  }
}
```

If circuit assets are not already in the workspace, copy them from an existing cache:

```bash
mkdir -p /tmp/prover-multi-gpu/gpu0/.work/galileo
cp -r /home/scroll/zzhang/scroll/.work/galileo/64cf16439a284e4449666c479a3ae42b568fea5e88610777ff6b5f2cda19d91182a47957139d0d1c1c3fbb28b9579c23f2823c0c6ff05669fe71ad4a0c92620e \
  /tmp/prover-multi-gpu/gpu0/.work/galileo/
```

### 7. Run the prover

OpenVM proving can overflow the default Rust thread stack, so set `RUST_MIN_STACK`. Run with no timeout:

```bash
cd /home/scroll/zzhang/scroll
RUST_MIN_STACK=33554432 ./target/release/prover \
  --config /tmp/chunk-6641451-export/prover-config.json \
  handle /tmp/chunk-6641451-export/handle-set.json
```

For a chunk of ~70 blocks, expect roughly 10–15 minutes on GPU.

### 8. Extract, verify, and save the proof

After the coordinator logs `proof verified and valid`, extract the proof from the shadow DB:

```bash
PGPASSWORD='shadow_pass' psql -h localhost -p 5433 \
  -U postgres -d shadow_rollup -Atq \
  -c "SELECT encode(proof, 'hex') FROM chunk WHERE index = 6641451;" \
  > /tmp/chunk-6641451-export/new-proof.hex

xxd -r -p /tmp/chunk-6641451-export/new-proof.hex > /tmp/chunk-6641451-export/new-proof.json

# independent verification
cd coordinator/build/bin
LD_LIBRARY_PATH=../internal/logic/libzkp/lib:$LD_LIBRARY_PATH \
  ./coordinator_tool verify chunk /tmp/chunk-6641451-export/new-proof.json
```

Save the final proof to a clear location:

```bash
cp /tmp/chunk-6641451-export/new-proof.json \
   /tmp/chunk-6641451/chunk_6641451_proof.json
```

## Common pitfalls

| Symptom | Cause | Fix |
|---------|-------|-----|
| `cannot execute UPDATE in a read-only transaction` | Coordinator connected to mainnet RDS with read-only user | Use shadow DB for coordinator writes |
| `server closed the connection unexpectedly` on `localhost:15432` | Mainnet RDS tunnel (stunnel) is down or broken | Retry later or ask ops to restart the tunnel |
| `record not found, uuid:...` during proof submission | Coordinator was restarted/killed after assigning the task; `prover_task` record was lost or not committed | Keep coordinator alive for the entire proving session; use `disable_timeout=true` |
| Prover aborts with stack overflow | Default Rust thread stack too small for OpenVM | `RUST_MIN_STACK=33554432` |
| S3 403 on circuit asset URLs | Directory listings are blocked or bucket is not public | Copy cached circuit assets into the prover workspace and set `debug_mode: true` |
| Query on `l2_block` times out | Table is huge; planner may seq-scan | `SET enable_seqscan = off;` and include `deleted_at IS NULL` |

## Cleanup

Stop the temporary coordinator when done. The shadow DB still contains the imported mainnet chunk; delete it if you no longer need it:

```bash
PGPASSWORD='shadow_pass' psql -h localhost -p 5433 \
  -U postgres -d shadow_rollup -Atq -c "
    DELETE FROM l2_block WHERE number BETWEEN 34180693 AND 34180761;
    DELETE FROM chunk WHERE index = 6641451;
  "
```
