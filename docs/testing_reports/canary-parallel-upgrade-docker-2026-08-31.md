# Canary Parallel-Upgrade Test Report (Docker Provers) — 2026-08-31

**Mode**: follow-mode canary parallel upgrade, second run — this time with **provers running as GPU docker containers** (bare metal was used only for coordinator/relayer/scripts). Extends the 2026-08-25 bare-metal run (`canary-parallel-upgrade-2026-08-25.md`).
**Host**: same machine, 2×RTX 4090, shadow DB local docker postgres:15 (`localhost:5433/shadow_rollup`), remote read-only `192.168.1.108:15432/mainnet_rollup`.
**Checkout**: `7d683b14` (branch `shadow-test-ai`, includes the Trap 38 halo2-gpu fixes and the Trap 41 `deleted_at IS NULL` sync fix — the latter measurably sped up baseline l2_block copies vs the previous run).

## Stacks under test

| | Old (production) | New (this checkout) |
|---|---|---|
| Guest | `79c1f8c` | zkvm-prover `bf887150` (v0.9.0) |
| bundle digest1 | `0x00398b78…9269` | `0x006770fb…5655` |
| Wrapper | `0x808297224e86b1a6055B5F790a2cE07Ed611f955` (forked MVRV) | deployed fresh per upgrade (see timeline) |
| Proof source | imported from remote DB (quarantine `remote_bundle_proof`) | **docker-proven locally** |

## Prover dockerization (new this run)

- Image: `scrolltech/prover:v4.7.14-c500b7d6-bf88715` (later `…-halo2gpu`), built locally with `build/dockerfiles/prover.Dockerfile` — the same production-style packaging as the devops `prover-image-build.yml` workflow (binary built externally, copied into a CUDA runtime image + solc).
- Launch: `04-prover-up.sh --docker` (driven by `30-canary-upgrade.sh --docker-provers` / `make canary-upgrade DOCKER_PROVERS=1 PROVER_IMAGE=…`). Each GPU gets a `shadow-prover-<gpu>` container: same-path work-dir mounts, `--network host`, `--gpus device=<n>`, `--user $(id -u):$(id -g)` with `HOME=/home/prover` and the SRS mounted at `/home/prover/.openvm/params` (Trap 12-proof). The container's **host PID** is written to the standard pidfile, so `11-follow-stop.sh` / `31-canary-rollback.sh` work unchanged; both also got a `docker rm -f shadow-prover-*` backstop.
- The devops workflow itself (`devops/core/prover_rust`, `init-repos.sh` pins `SCROLL_COMMIT`) was used as reference only; no CI trigger was needed.

## Timeline (UTC)

| Time | Event |
|---|---|
| 05:52 | First Phase A attempt failed fast: script default `MAINNET_DSN` is `localhost:15432` (RDS-tunnel convention) but this host reaches the DB at `192.168.1.108:15432` directly. Fixed via env override (`local-secrets.md` already had the right value). |
| 05:55 | Phase A up: `10-follow-up.sh --import-proofs --reset`. Fork + baseline + 18,673 production proofs quarantined. No coordinator/provers by design. |
| 05:56–05:58 | **Phase A acceptance ✅**: bundles 18786–18790 (ends 519531–519538) finalized purely from imported production proofs, zero local proving. |
| 06:00 | **t1**: `make canary-upgrade DOCKER_PROVERS=1` — boundary N=519539, new wrapper `0x3a251c0c…5577` deployed + registered via genuine `updateVerifier(10, 519539, …)`, routing asserted both sides, digest-differ check passed. **Docker prover start failed** (Finding 1); verifier/boundary/coordinator steps had already succeeded. |
| 06:01 | Finding 1 fixed (`04-prover-up.sh` canonicalizes `work_dir`); CANARY_* keys recorded manually (step j had been skipped on failure); docker provers started directly — both healthy, GPU-active in-container within a minute. |
| 06:04 | **Rollback drill ✅**: `31-canary-rollback.sh` — docker provers stopped cleanly via pidfile host-PID, `docker rm -f` backstop removed containers, `updateVerifier(10, 519539, OLD_WRAPPER)` restored, boundary cleared to +∞. |
| 06:12 | **Re-upgrade ✅**: N unchanged (519539), new wrapper `0xA7898f4B…c3a` registered; docker provers restarted by the script itself — idempotent docker path confirmed. |
| 06:56 | Mainnet produced bundle 18791 (end 519539 — first ≥ N). |
| 07:29–07:30 | **t2 ✅**: bundle 18791 proven by a docker prover and finalized through the new wrapper (tx `0x59a75b24…ee4f1` status true; fork `lastFinalizedBatchIndex` → 519539; proof digest1 `0x006770fb…5655` = new guest, not imported). |
| 07:41 | Switched image to the **halo2-gpu** build (`make prover_halo2gpu`, tag `…-halo2gpu`) after noticing 18791's bundle SNARK ran on CPU (Finding 4); provers restarted, one stale coordinator-side task assignment cleared. |
| 07:49–08:00 | **GPU-SNARK bundle ✅**: bundle 18792 (end 519540) proven by `shadow-prover-1` — halo2 wrapper `Create EVM proof` **6.05 s**, GPU mem peak **8.8 GiB** (Trap 38 fixes hold in a dockerized mixed-type process); finalized 08:00:02 via the new wrapper (tx `0x57e4bfe7…318c`, digest verified). Fork `lastFinalizedBatchIndex` → 519540. |

## Bundle proving time: CPU vs GPU SNARK

| Bundle | Image | Bundle task span | halo2 wrapper `Create EVM proof` |
|---|---|---|---|
| 18791 | `cuda`-only (`make prover`) | 06:56 → 07:29 ≈ **33 min** | dominates (CPU) |
| 18792 | `halo2-gpu` (`make prover_halo2gpu`) | ~07:49 → 07:59 ≈ **10 min** (STARK agg layers now dominate) | **6.05 s** |

Rule of thumb confirmed: if a bundle task takes tens of minutes, check whether the binary was built with `make prover_halo2gpu`.

## Findings / fixes this run

1. **Docker mode broke on the `lib/..` path component** (fixed): `04-prover-up.sh` built `work_dir` as `${SCRIPT_DIR}/../.work/prover-N`. dockerd path-cleans bind-mount targets, but the prover opens its config by the literal path — inside the container `/…​/lib/..` does not resolve (`lib` doesn't exist there) → `File::open` ENOENT at `prover.rs:182`. Fixed by canonicalizing `work_dir` (`cd … && pwd`) before generating configs/mounts. Harmless on bare metal, fatal in docker.
2. **`MAINNET_DSN` script default is `localhost:15432`**; on this host the DB is at `192.168.1.108:15432` (no local tunnel). Export `MAINNET_DSN` explicitly (the scripts honor the env var).
3. **Restarting a prover leaves its coordinator-side assignment** ("already assigned a task" on the new process): cleared via `DELETE FROM prover_task WHERE proving_status IN (1,3)` (the sweeper would also reset it within ~10 min).
4. **Image built from a `make prover` binary runs the bundle SNARK on CPU** — 33 min vs 6 s. For 24 GB-class GPUs always build the image from a `make prover_halo2gpu` binary. GPU contention check before the swap: only the two `shadow-prover-*` containers were on the cards.
5. The Trap 41 `deleted_at IS NULL` predicates (previous commit) visibly sped up the baseline: l2_block range copies that took minutes per cycle on 2026-08-27 completed promptly during this run's baseline.

## Acceptance summary

| Criterion | Status |
|---|---|
| Phase A: ≥3 backlog bundles finalized purely from imported production proofs | ✅ (18786–18790, five bundles) |
| t1: genuine `updateVerifier`, routing asserted, digest differ | ✅ (N=519539) |
| Provers run in docker (no bare-metal prover) | ✅ (`shadow-prover-0/1`, GPU-active in-container) |
| Docker prover stop/start through 31/30 paths | ✅ rollback drill + re-upgrade |
| t2: first ≥N bundle proven by docker prover, finalized via new wrapper | ✅ (18791, digest-verified) |
| halo2-gpu bundle in docker (Trap 38 fixes validated in containers) | ✅ (18792: 6.05 s SNARK, peak 8.8 GiB) |

Note: unlike the 2026-08-25 run, no < N bundle remained unfinalized at t1 (mainnet bundle boundary aligned exactly), so the interleaved-old-after-upgrade step had no specimen this time; it remains covered by the previous run and by the rollback drill.
