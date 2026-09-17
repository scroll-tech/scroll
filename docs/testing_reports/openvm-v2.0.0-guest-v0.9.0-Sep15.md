# E2E Test Report: OpenVM v2.0 + Guest Assets v0.9.0

**Date:** 2026-09-15
**PR:** #1816
**Branch:** `feat/zkvm-v0.9.0-openvm2` (tested via the superset branch `shadow-test-ai`)
**OpenVM Version:** v2.0.0
**Guest Assets Version:** v0.9.0 (zkvm-prover `bf887150` = tag `v0.9.0` + halo2-gpu commit)
**Asset URL:** `s3://circuit-release/scroll-zkvm/releases/v0.9.0/`

---

## Test methodology

The upgrade was exercised end-to-end on a **shadow fork of mainnet** (anvil L1 fork + shadow
coordinator + local provers), in "canary parallel-upgrade" mode: old (production, guest `79c1f8c`)
and new (v0.9.0) proving ran side by side, with the multi-verifier rollup contract routing batch
finalization to the new wrapper only from an upgrade boundary N onward. Production bundle proofs
were imported from the mainnet DB for the pre-upgrade backlog; all post-boundary bundles were
proven locally by the v0.9.0 stack.

Full run logs and acceptance tables: `canary-parallel-upgrade-2026-08-25.md` (bare metal) and
`canary-parallel-upgrade-docker-2026-08-31.md` (dockerized provers) in this directory.

## Environment

| Component | Version / Details |
|-----------|-------------------|
| Hardware | 2× NVIDIA RTX 4090 (24 GB) |
| CUDA | 12.8 (host), 12.9.1 runtime image |
| Rust Toolchain | nightly-2025-11-20 |
| Solc | 0.8.24 |
| Prover | `target/release/prover` (`--features halo2-gpu`), also dockerized via `build/dockerfiles/prover.Dockerfile` |
| Coordinator | `coordinator_api` on port 8390, shadow DB postgres:15 |

## Test results

- **Phase A (backlog)**: bundles 18786–18790 finalized purely from imported production proofs —
  proves the new coordinator/wire format stays compatible with the old proof pipeline up to the
  upgrade boundary.
- **t1 (upgrade boundary)**: genuine `updateVerifier(10, 519539, new_wrapper)` on the forked
  rollup contract; pre-/post-boundary routing asserted; bundle digests of old vs new guest
  confirmed different (`0x00398b78…9269` vs `0x006770fb…5655`).
- **Rollback drill**: provers stopped, `updateVerifier(10, 519539, old_wrapper)` restored, then
  re-upgrade — idempotent.
- **t2 (first post-boundary bundle)**: bundle 18791 (end batch 519539) proven by the new stack
  and finalized through the new wrapper (tx `0x59a75b24…ee4f1`), digest-verified as new-guest.
- **halo2-gpu SNARK**: bundle 18792 finalized (tx `0x57e4bfe7…318c`) with the halo2 wrapper
  `Create EVM proof` phase at **6.05 s**, GPU memory peak **8.8 GiB**.

## Proving time: CPU vs GPU bundle SNARK

| Bundle | Build | Bundle task span | halo2 `Create EVM proof` |
|---|---|---|---|
| 18791 | `make prover` (`cuda`) | ≈ 33 min | dominates (CPU) |
| 18792 | `make prover_halo2gpu` | ≈ 10 min (STARK layers now dominate) | 6.05 s |

Rule of thumb: if a bundle task takes tens of minutes on a 24 GB GPU, the binary was not built
with `make prover_halo2gpu`.

## Issues found during testing (fixed in this PR / companion commits)

1. **VRAM starvation in mixed-task provers** — a prover holding chunk+batch+bundle handlers
   exhausted 24 GB when the halo2 SNARK phase started. Fixed by releasing child GPU SDKs after
   deferral setup (`deferral` module) and by downloading the pre-built `agg_vk.bin` instead of
   deriving the aggregation VK on GPU.
2. **`def_hook_commit` undefined for bundle tasks** — bundle proving additionally needs
   batch-over-chunk deferral initialized, not just bundle-over-batch.
3. **CPU-SNARK image pitfall** — the first docker image was built from a `make prover` binary;
   bundle SNARK silently ran on CPU (33 min). Documented; production images should be built from
   `make prover_halo2gpu`.

## Upgrade-compatibility notes (operational)

- The v0.9.0 `StarkProof` wire format (`proof` / `user_pvs_proof`) is **not** backward
  compatible: provers older than v4.8.0 are rejected at login (`min_prover_version`), and
  in-flight v0.8.0 proofs stored in the coordinator DB no longer deserialize — drain or reset
  in-flight batch/bundle tasks before deploying.
- Coordinator verifier assets now **require `agg_vk.bin`** (batch circuit's aggregation VK) in
  the assets dir; the coordinator refuses to start without it. `setup_releases.sh` downloads it.
