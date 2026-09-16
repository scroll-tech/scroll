# Canary Parallel-Upgrade Test Report (Prebuilt CI Image + Production-Deployed Verifier) — 2026-09-16

**Mode**: follow-mode canary parallel upgrade, third run — **happy path only** (no rollback drill; new policy: canary validates the straightforward upgrade, the rollback path remains available but is not exercised every run). Extends the 2026-08-31 docker run (`canary-parallel-upgrade-docker-2026-08-31.md`) and the 2026-09-11 run (`canary-parallel-upgrade-2026-09-11.md`).
**Host**: same machine, 2×RTX 4090 — **GPU 0 busy the whole run** (unrelated ComfyUI service, 12.5 GiB); all proving on **GPU 1 only** (`GPUS=1`). Shadow DB local docker postgres:15 (`localhost:5433/shadow_rollup`), remote read-only `192.168.1.108:15432/mainnet_rollup`.
**Checkout**: `24d8ab7b` (branch `shadow-test-ai`), which includes `eec4ed75` — the merge of PR #1816 (`feat/zkvm-v0.9.0-openvm2`, `8cc4b0dd`, OpenVM v2 / guest v0.9.0 / workspace v4.8.0) into `shadow-test-ai`.

## What is new this run

1. **No locally built prover, no bare-metal prover.** The new stack runs the **prebuilt CI image** `scrolltech/cuda-prover:v4.8.0-8cc4b0dd-bf88715-` (digest `sha256:0bab1629…`), built by the devops `prover-image-build.yml` workflow from the PR #1816 HEAD (confirmed: the workflow runs `make prover_halo2gpu` — halo2-gpu SNARK confirmed in practice, see timeline).
2. **No fresh verifier deployment.** t1 registered the **production-deployed v0.9.0 wrapper** `0x7966FE0De13e3B81533470640Ac489223476B254` (already on mainnet, hence present on the anvil fork) via the genuine `updateVerifier` path, using the new `--reuse-wrapper` flag of `30-canary-upgrade.sh`. Pre-flight correctness check against mainnet: `verifierDigest1()` = `0x006770fb…5655` and `verifierDigest2()` = `0x00488b19…33f7` — both byte-identical to S3 `releases/v0.9.0/bundle/digest_{1,2}.hex`; `protocolVersion()` = 10; `plonkVerifier()` = `0x0d2A59fd…01C6`. The script additionally re-checks on-chain digest1 == S3 digest1 at t1.
3. **No rollback drill** — per the new policy the canary is the happy-path upgrade rehearsal only.

## Stacks under test

| | Old (production) | New (PR #1816) |
|---|---|---|
| Guest | `79c1f8c` (production) | zkvm-prover `bf887150` (v0.9.0) |
| bundle digest1 | `0x00398b78…9269` | `0x006770fb…5655` |
| Wrapper | `0x808297224e86b1a6055B5F790a2cE07Ed611f955` (forked MVRV) | **`0x7966FE0D…B254` (mainnet-deployed, reused)** |
| Proof source | imported from remote DB (`remote_bundle_proof`) | **docker-proven locally** (`shadow-prover-1`, GPU 1) |
| Coordinator | — | v4.8.0, built from this checkout (`eec4ed75`) |

## Timeline (UTC)

| Time | Event |
|---|---|
| 08:52:50 | Phase A up: `10-follow-up.sh --import-proofs --reset` (MAINNET_DSN pointed at `192.168.1.108:15432`). Fork + baseline + 18,924 production proofs quarantined. No coordinator/provers by design. |
| 08:53:40–08:54:10 | **Phase A acceptance ✅**: bundles 19039–19041 (ends 519830–519832) finalized purely from imported production proofs — `lastFinalizedBatchIndex` 519829 → 519832, zero local proving. |
| 08:56 | **t1**: `make canary-upgrade GPUS=1 DOCKER_PROVERS=1 REUSE_WRAPPER=1 PROVER_IMAGE=scrolltech/cuda-prover:v4.8.0-8cc4b0dd-bf88715-`. Boundary N=519833 (= only unproven bundle 19042). Genuine `updateVerifier(10, 519833, 0x7966FE0D…B254)` — tx `0x52532e12…c919`. Routing asserted both sides (`getVerifier(10, 519832)` → old wrapper, `(10, 519833)` → new), digest-differ check passed, on-chain digest1 == S3 digest1 ✅. Coordinator (v4.8.0) + `shadow-prover-1` container started. |
| 08:57–09:05 | Bundle 19042 (1 batch / 4 chunks) proven by `galileo6-shadowfork-prover-1`: chunks ≈ 66.7 s each (`universal proving speed: 5.60MHz`), then batch, then bundle STARK agg. |
| 09:06:00–09:08:02 | Bundle task span (coordinator `prover_task` row, type 3). halo2-gpu SNARK `Create EVM proof` **1.541 s** — the prebuilt image is a halo2-gpu build. |
| 09:08 | **t2 ✅**: bundle 19042 (end 519833, first ≥ N) finalized — tx `0xc5f6ea5c…edc9` status `true`; `lastFinalizedBatchIndex` → 519833; `rollup_status=5` at 09:09. `cast run` trace: `ScrollChain.finalizeBundle` → `MVRV.verifyBundleProof(10, 519833, …)` → **`0x7966FE0D…B254::verify` [staticcall]** — the production-deployed wrapper verified the locally generated v0.9.0 proof. |
| 09:09+ | Parallel period continues: next bundle's chunks already proving on GPU 1; remote proofs ≥ N keep accumulating in the quarantine table (rollback remains possible, just not drilled). |

Note: t2 came ~12 min after t1 because the boundary landed exactly on the frontier — bundle 19042 contained a single fresh batch, so only 4 chunks + 1 batch + 1 bundle needed proving. Larger backlogs cost proportionally more (per CURRENT-STACK sizing: chunk ~2 min, bundle end-to-end ~10–15 min on one 4090).

## Acceptance summary

| Criterion | Status |
|---|---|
| Phase A: ≥3 backlog bundles finalized purely from imported production proofs | ✅ (19039–19041) |
| t1: genuine `updateVerifier`, routing asserted, digest differs from old | ✅ (N=519833) |
| t1: reused wrapper bound to the release under test (on-chain digest1 == S3) | ✅ |
| New prover = prebuilt CI docker image (no bare metal, no local build) | ✅ (`shadow-prover-1`, GPU-active in-container) |
| t2: first ≥N bundle proven by the docker prover, finalized via the **production-deployed** wrapper | ✅ (19042; trace shows `0x7966FE0D…B254::verify`) |
| halo2-gpu SNARK inside the prebuilt image | ✅ (1.541 s) |
| Rollback drill | ⏭️ skipped by policy (happy-path-only canary) |

As in the previous two runs, no < N bundle remained unfinalized at t1 (boundary aligned with the mainnet frontier), so the interleaved-old-after-upgrade case had no specimen; it remains covered by the 2026-08-25 run.

## Proving-time comparison (old production guest vs new v0.9.0)

Old = production stack (guest `79c1f8c`, OpenVM 1.x) — `proof_time_sec` from the
mainnet rollup DB, last 500 chunks / 500 batches / 100 bundles (coordinator
wall time per task, includes dispatch overhead; p50 given to damp outliers).
New = this run's docker prover on a single RTX 4090 — 9 chunks / 2 batches /
2 bundles (small sample, but very tight variance: 80–81 s / 21 s / 121–122 s).
Production bundles averaged **1.2 batches/bundle** over the sample window, so
the bundle row is a like-for-like comparison (our bundles had 1 batch).

| Level | Old avg (p50) | New avg | Speedup (avg) |
|---|---|---|---|
| Chunk | 567.6 s (490 s) | 80.1 s | **7.1×** (6.1× @ p50) |
| Batch | 121.6 s (101 s) | 21.0 s | **5.8×** (4.8× @ p50) |
| Bundle | 423.5 s (400 s) | 121.5 s | **3.5×** (3.3× @ p50) |

Caveats: old-stack max chunk time was 5789 s (retries/queueing noise) — the
p50 column is the fairer baseline; the new sample is small and from one GPU.

## Findings / fixes this run

1. **Makefile flag concatenation bug** (fixed in `24d8ab7b`): `make canary-upgrade DOCKER_PROVERS=1 REUSE_WRAPPER=1` emitted `--docker-provers--reuse-wrapper`. Fixed by adding the trailing space inside the first `$(if …)`.
2. **`mainnet-next.json` is git-ignored** (generated from `.json.template`): the reused wrapper address must be set in the local generated config before t1 (`.contracts.deployed_verifier`), not committed. The GUIDE's canary section now documents this.
3. **PR #1816 merged into `shadow-test-ai`** (`eec4ed75`): 11 conflicts, all resolved to the PR side (it is the reviewed final of the same v0.9.0 work the branch carried as WIP), plus one genuine merge artifact fixed (`mod deferral;` duplicated in `prover-bin/src/main.rs`). Post-merge `go build` (common/coordinator/rollup) and `cargo check -p prover` clean.
4. **GPU 0 contention**: an unrelated ComfyUI service holds 12.5 GiB on GPU 0 permanently. Single-GPU (`GPUS=1`) canary runs are the default on this host from now on; 1×4090 keeps pace with mainnet cadence (also confirmed 2026-09-11).
5. The prebuilt `cuda-prover` image is a `make prover_halo2gpu` build — verified both from the workflow source and empirically (1.541 s SNARK). The 2026-08-31 "33 min CPU SNARK" trap does not apply to CI-built images.
