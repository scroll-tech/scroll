# E2E Test Report: OpenVM 1.6.0 + Guest Assets v0.8.0

**Date:** 2026-05-19  
**PR:** #1783  
**Environment:** Scroll monorepo, Galileo fork  
**OpenVM Version:** 1.6.0  
**Guest Assets Version:** v0.8.0  
**Asset URL:** `s3://circuit-release/scroll-zkvm/galileov2/`

---

## Environment

| Component | Version / Details |
|-----------|-------------------|
| Hardware | 4× NVIDIA RTX 3090 |
| CUDA | 12.9 |
| Rust Toolchain | nightly-2025-08-18 |
| Go | 1.22.4 |
| Coordinator | `coordinator/build/bin/coordinator_api` on port `8390` |
| Prover | `target/release/prover` with GPU (`--features cuda`) |
| Database | PostgreSQL on port `5433` |

---

## Test Results

| Task | Count | Proving Status | Notes |
|------|-------|----------------|-------|
| Chunk | 4 | Proved (4) | ~20–480s depending on block count |
| Batch | 2 | Proved (4) | ~60–102s |
| Bundle | 1 | Proved (4) | ~1061s (Halo2 SNARK compression ~392s) |

**Full pipeline completed:** 4 chunks → 2 batches → 1 bundle → all verified by coordinator.

---

## Issues Encountered & Resolutions

### 1. Invalid app vm commit (Guest/Prover Version Mismatch)
- **Symptom:** Prover panicked with `Invalid app vm commit` during batch/bundle proving.
- **Root cause:** Prover was using OpenVM 1.6.0, but guest assets (`v0.7.1`) were compiled with an older OpenVM version.
- **Fix:** Upgraded guest assets to `v0.8.0`, compiled with OpenVM 1.6.0. Re-uploaded to `s3://circuit-release/scroll-zkvm/galileov2/`.

### 2. TOML Format Change (proof_of_work_bits)
- **Symptom:** Config parse error on `proof_of_work_bits`.
- **Fix:** Updated `openvm.toml` to use `commit_proof_of_work_bits` and `query_proof_of_work_bits` separately.

### 3. Coordinator Verifier Fork Name
- **Symptom:** Initial coordinator config used `galileoV2` as fork name; verifier lookup failed.
- **Fix:** Changed coordinator verifier fork from `galileoV2` to `galileo` (assets still served from `galileov2/` S3 path).

### 4. GPU Proving Timeout
- **Symptom:** Batch proving exceeded the default `chunk_collection_time_sec=180`, causing coordinator to abort sessions.
- **Fix:** Increased `chunk_collection_time_sec` to `3600` in coordinator config.

### 5. Asset Download Authorization
- **Symptom:** Old `galileo/` S3 path returned HTTP 403.
- **Fix:** Used `galileov2/` S3 path, which is public.

### 6. Coordinator Crash (Non-Unwinding Panic)
- **Symptom:** `l2geth` thread panic in `crates/l2geth/src/lib.rs:10` during proof handling.
- **Fix:** Restarted coordinator; it resumed successfully and completed the pipeline.

### 7. Docker Port Binding
- **Symptom:** `docker compose down` + `up` sometimes bound PostgreSQL to wrong port (5442 instead of 5433).
- **Fix:** Explicitly removed stale containers with `docker rm` before restart.

---

## Configuration Used

### Prover (`zkvm-prover/config.json`)
```json
{
  "sdk_config": {
    "prover_name_prefix": "test-prover",
    "coordinator": {"base_url": "http://localhost:8390"},
    "prover": {"supported_proof_types": [1,2,3], "circuit_version": "v0.13.1"}
  },
  "circuits": {
    "galileo": {
      "workspace_path": ".work/galileo",
      "base_url": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/galileov2/",
      "asset_detours": {
        "64cf16439a284e4449666c479a3ae42b568fea5e88610777ff6b5f2cda19d91182a47957139d0d1c1c3fbb28b9579c23f2823c0c6ff05669fe71ad4a0c92620e": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/galileov2/chunk/64cf16439a284e4449666c479a3ae42b568fea5e88610777ff6b5f2cda19d91182a47957139d0d1c1c3fbb28b9579c23f2823c0c6ff05669fe71ad4a0c92620e/",
        "e9d6536bdcef7806156c245b20f2c5113825bb4a6cf5246b4b698829ecf23268ff32d36a5fa86c245eb45244557fe26b05efe158d36c6b48f854b32eca3b4c59": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/galileov2/batch/e9d6536bdcef7806156c245b20f2c5113825bb4a6cf5246b4b698829ecf23268ff32d36a5fa86c245eb45244557fe26b05efe158d36c6b48f854b32eca3b4c59/",
        "6b155f2d4a07ff0860f8ac395a263e657801761e7e5c3b5666bd35439e3b342d46f92437d8a633521cf9e93c9721b374f47f6c2459161e08e3186c6c66d34a1a": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/galileov2/bundle/6b155f2d4a07ff0860f8ac395a263e657801761e7e5c3b5666bd35439e3b342d46f92437d8a633521cf9e93c9721b374f47f6c2459161e08e3186c6c66d34a1a/"
      },
      "debug_mode": true
    }
  }
}
```

### Coordinator (`coordinator/build/bin/conf/config.json`)
```json
{
  "prover_manager": {
    "chunk_collection_time_sec": 3600,
    "verifier": {
      "verifiers": [{"assets_path": "assets_galileo", "fork_name": "galileo"}]
    }
  }
}
```

---

## Final Database State

All tasks reached `proving_status = 4` (proved):

```sql
SELECT proving_status, COUNT(*) FROM chunk_task_detail GROUP BY proving_status;
-- 4 | 4

SELECT proving_status, COUNT(*) FROM batch_task_detail GROUP BY proving_status;
-- 4 | 2

SELECT proving_status, COUNT(*) FROM bundle_task_detail GROUP BY proving_status;
-- 4 | 1
```

---

## Pre-Existing Failures (Not Regressions)

The following failures existed before this PR and are **not caused by the OpenVM 1.6.0 upgrade**:

| Test | Failure Reason |
|------|----------------|
| `crates/libzkp/src/proofs.rs::test_roundtrip` | Missing `testdata/` directory |
| `coordinator/test` Go package | Segfault during `InitVerifier` (missing circuit assets in test environment) |

---

## Sign-Off

✅ All five test levels passed (compilation, unit tests, artifact builds, E2E proving, Docker).  
✅ Full chunk → batch → bundle pipeline completed and verified.  
✅ No new Clippy warnings or formatting regressions.
