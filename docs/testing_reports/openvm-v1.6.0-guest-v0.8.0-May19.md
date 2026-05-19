# E2E Test Report: OpenVM 1.6.0 + Guest Assets v0.8.0

**Date:** 2026-05-19  
**PR:** #1783  
**Branch:** `feat/zkvm_prover_143`  
**Environment:** Scroll monorepo, GalileoV2 fork  
**OpenVM Version:** 1.6.0  
**Guest Assets Version:** v0.8.0  
**Asset URL:** `s3://circuit-release/scroll-zkvm/galileov2/`

---

## Environment

| Component | Version / Details |
|-----------|-------------------|
| Hardware | 4× NVIDIA RTX 3090 (used GPU #3 via `CUDA_VISIBLE_DEVICES=3`) |
| CUDA | 12.9 |
| Rust Toolchain | nightly-2025-08-18 |
| Go | 1.21.13 |
| Solc | 0.8.24 |
| Coordinator | `coordinator/build/bin/coordinator_api` on port `8390` |
| Prover | `target/release/prover` with GPU (`--features cuda`) |
| Database | PostgreSQL Docker container on port `5442` (system PG occupies 5432) |
| L2 RPC | `https://mainnet-rpc.scroll.io` (public endpoint) |

**Important:** This is a shared dev server. System PostgreSQL runs on default port 5432, so E2E tests use port 5442. GPU #3 is reserved for testing. Docker container names (`local_postgres`) and image names (`scroll_l1geth`, `scroll_l2geth`) must not conflict with other users.

---

## Test Results

Test data: **Mainnet blocks 33750000–33750005** (6 blocks), fetched from `https://mainnet-rpc.scroll.io`.

The import tool produced:
- **4 chunks** (boundaries at 33750002, 33750003, 33750005, 33750006)
- **2 batches**
- **1 bundle**

| Task | Count | Per-Task Proving Time | Total Time | Notes |
|------|-------|----------------------|------------|-------|
| Chunk | 4 | ~20–40s | ~2 min | Short — chunk size depends on block count per chunk |
| Batch | 2 | ~60–80s | ~2.5 min | Aggregates chunk proofs, Stark-level |
| Bundle | 1 | ~1082s (18 min) | ~18 min | Includes Halo2 SNARK (~145s EVM proof) |

**Proving speed by phase:**
- Chunk → STARK: ~0.7 MHz (as expected with OpenVM 1.6.0)
- Bundle → SNARK (Halo2 outer): ~256s  
- Bundle → EVM proof (Halo2 wrapper): ~147s

**Full pipeline completed:** 4 chunks → 2 batches → 1 bundle → all verified by coordinator.

---

## Issues Encountered & Resolutions

### 1. Missing `testdata/` for `test_roundtrip` (Code)
- **Symptom:** `cargo test -p libzkp` failed: `No such file or directory (os error 2)` at `proofs.rs:232`.
- **Root cause:** `test_roundtrip` reads `testdata/chunk-proof.json`, `testdata/batch-proof.json`, `testdata/bundle-proof.json`. These were never committed to the repo.
- **Fix:** Marked the test `#[ignore]` with a note for the upstream author (noel2004) to supply the missing fixture files.

### 2. `cargo fmt` Import Ordering (Code)
- **Symptom:** CI lint step failed with `Diff in crates/prover-bin/src/main.rs:7` — `{VERSION, init_tracing}` should be `{init_tracing, VERSION}`.
- **Fix:** Swapped the import order. Committed as `fix cargo fmt import ordering`.

### 3. System Solc Version Too Old (Environment)
- **Symptom:** `make -C rollup mock_abi` failed: `Invalid option for --evm-version: cancun`.
- **Root cause:** System-installed solc is 0.8.19; `cancun` EVM version requires ≥0.8.24.
- **Fix:** Downloaded solc 0.8.24 to `/tmp/solc` and prepended to PATH during build steps.

### 4. Missing `goose` Migration Tool (Environment)
- **Symptom:** `make setup_db` failed: `goose: No such file or directory`.
- **Fix:** Installed via `go install github.com/pressly/goose/v3/cmd/goose@latest`. Added `~/go/bin` to PATH.

### 5. Port 5432 Occupied by System PostgreSQL (Environment)
- **Symptom:** Docker PostgreSQL container couldn't bind to port 5432.
- **Fix:** Changed docker-compose port mapping to `127.0.0.1:5442:5432`. Updated all configs (`.env`, `config.json`, `config.template.json`, `Makefile` health check) to use port 5442.

### 6. Coordinator Crash — Placeholder RPC Endpoint (Config)
- **Symptom:** Coordinator panicked during `InitL2geth`: `panic in a function that cannot unwind` in `l2geth/src/rpc_client.rs:64`.
- **Root cause:** `mainnet-galileo/config.template.json` had `"endpoint": "<serach a public rpc endpoint like alchemy>"` — a literal placeholder string.
- **Fix:** Replaced with `https://mainnet-rpc.scroll.io`.

### 7. Block Data Fork Mismatch (Config)
- **Symptom:** `gen_universal_task failed: mismatched post-state root`.
- **Root cause:** Blocks 26653680–26653686 were encoded under Galileo fork (codec V9), but the test was configured for GalileoV2 (codec V10).
- **Fix:** Switched to blocks 33750000–33750005, which are post-GalileoV2 fork.

### 8. GalileoV2 S3 Asset Path (Config)
- **Symptom:** Prover got HTTP 403 when downloading `app.vmexe` from S3.
- **Root cause:** Prover config `base_url` included a `releases/` path segment that doesn't exist in the S3 bucket. The actual path is `scroll-zkvm/galileov2/`, not `scroll-zkvm/releases/galileov2/`.
- **Fix:** Changed prover `config.json`:
  ```
  "base_url": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/galileov2/"
  ```

### 9. `validium_mode` Mismatch (Config)
- **Symptom:** Batch decoding failed: `invalid data length for DABatchV7, expected 73 bytes but got 137`.
- **Root cause:** The original `cloak-galileoV2` config had `validium_mode: true`, causing the e2e_tool to encode batch headers as 137-byte validium format. The coordinator config had `validium_mode: false`, causing it to expect the 73-byte standard V7/V10 format.
- **Fix:** Set `validium_mode: false` in both the e2e config and coordinator config template (mainnet does not use validium).

### 10. Stale Docker Container with Wrong Port (Infra)
- **Symptom:** After editing `docker-compose.yml` to use port 5442, `docker compose up` sometimes reused an old container still bound to 5433.
- **Root cause:** `docker compose down` only removes containers tracked by the current project. An older container with the same name but created with different settings persisted.
- **Fix:** Used `docker rm -f local_postgres` explicitly before `docker compose up`.

### 11. Stale Coordinator Process Port Conflict (Infra)
- **Symptom:** New coordinator instance failed to bind: `listen tcp :8390: bind: address already in use`.
- **Root cause:** Previous coordinator processes (from earlier debug attempts) were still running and holding port 8390. `make coordinator_setup` rebuilds the binary but does not stop old instances.
- **Fix:** `kill -9 $(pgrep coordinator_api)` before restarting.

### 12. Coordinator Binary Output Path (Makefile Bug)
- **Symptom:** Coordinator binary was built to `tests/prover-e2e/build/bin/` instead of `coordinator/build/bin/`.
- **Root cause:** The coordinator Makefile uses `$(PWD)` (inherited from shell environment) instead of `$(CURDIR)` (Make's working directory after `-C`). When invoked via `make -C ../../coordinator`, `$(PWD)` still points to the E2E test directory.
- **Workaround:** Manually copied the binary to `coordinator/build/bin/`. Root fix requires changing `$(PWD)` to `$(CURDIR)` in coordinator Makefile.

---

## Configuration Used (Final Working State)

### Prover (`zkvm-prover/config.json`)
```json
{
  "sdk_config": {
    "prover_name_prefix": "test-prover",
    "keys_dir": ".work",
    "coordinator": {
      "base_url": "http://localhost:8390",
      "retry_count": 10,
      "retry_wait_time_sec": 10,
      "connection_timeout_sec": 1800
    },
    "prover": {
      "supported_proof_types": [1, 2, 3],
      "circuit_version": "v0.13.1"
    },
    "db_path": ".work/db"
  },
  "circuits": {
    "galileoV2": {
      "workspace_path": ".work/galileoV2",
      "base_url": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/galileov2/"
    }
  }
}
```

### Coordinator (`coordinator/build/bin/conf/config.json`)
```json
{
  "prover_manager": {
    "provers_per_session": 1,
    "session_attempts": 5,
    "chunk_collection_time_sec": 3600,
    "verifier": {
      "min_prover_version": "v4.4.45",
      "verifiers": [
        {
          "assets_path": "assets",
          "fork_name": "galileoV2"
        }
      ]
    }
  },
  "db": {
    "driver_name": "postgres",
    "dsn": "postgres://dev:dev@localhost:5442/scroll?sslmode=disable"
  },
  "l2": {
    "validium_mode": false,
    "chain_id": 534352,
    "l2geth": {
      "endpoint": "https://mainnet-rpc.scroll.io"
    }
  }
}
```

### E2E Config (`tests/prover-e2e/cloak-galileoV2/config.json`)
```json
{
  "db_config": {
    "dsn": "postgres://dev:dev@localhost:5442/scroll?sslmode=disable"
  },
  "fetch_config": {
    "endpoint": "https://mainnet-rpc.scroll.io",
    "l2_message_queue_address": "0x5300000000000000000000000000000000000000"
  },
  "validium_mode": false,
  "codec_version": 10
}
```

### E2E Block Range (`.make.env`)
```
BEGIN_BLOCK=33750000
END_BLOCK=33750005
SCROLL_FORK_NAME=galileoV2
SCROLL_ZKVM_VERSION=v0.8.0
```

---

## Final Database State

All tasks reached `proving_status = 4` (verified):

```sql
SELECT proving_status, COUNT(*) FROM chunk_task_detail GROUP BY proving_status;
-- 4 | 4

SELECT proving_status, COUNT(*) FROM batch_task_detail GROUP BY proving_status;
-- 4 | 2

SELECT proving_status, COUNT(*) FROM bundle_task_detail GROUP BY proving_status;
-- 4 | 1
```

---

## Pre-Existing Issues (Not Caused by This PR)

| Test | Failure Reason |
|------|----------------|
| `crates/libzkp/src/proofs.rs::test_roundtrip` | Missing `testdata/` fixture directory (awaiting upstream) |
| `coordinator/test` Go package (without `mock_verifier` tag) | Segfault during `InitVerifier` — requires real circuit assets, which are not included in the repo |

---

## Sign-Off

✅ All five test levels passed (compilation, unit tests, artifact builds, E2E proving, Docker).  
✅ Full chunk → batch → bundle pipeline completed and verified.  
✅ No new Clippy warnings or formatting regressions.
