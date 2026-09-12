# Canary Parallel-Upgrade Test Report — 2026-09-11

**Mode**: follow-mode canary parallel upgrade, third run (first: 2026-08-25 bare-metal, second: 2026-08-31 docker). Full cycle this time: Phase A → staging → t1 → parallel window → rollback drill → post-rollback acceptance → re-upgrade → t2 → soak (2 new-stack bundles).
**Host**: same machine, 2×RTX 4090 — but **GPU0 was occupied by an unrelated process (ComfyUI, 22.4 GiB)**, so the entire run used **GPUS=1 (GPU1 only)**. Shadow DB local docker postgres:15 (`localhost:5433/shadow_rollup`), remote read-only `192.168.1.108:15432/mainnet_rollup`.
**Checkout**: `403a1e9e` (branch `shadow-test-ai`; prover binary from 8-31 `prover_halo2gpu` build — the commits since are scripts/docs only).

## Stacks under test

| | Old (production) | New (this checkout) |
|---|---|---|
| Guest | unchanged since 8-31 (wrapper digest1 `0x00398b78…9269`) | zkvm-prover `bf887150` (S3 `releases/v0.9.0`, digest1 `0x006770fb…5655`) |
| Wrapper | `0x808297224e86b1a6055B5F790a2cE07Ed611f955` (forked MVRV) | `0x3a251c0c…5577` (t1, N=519738) then `0xA7898f4B…c3a` (re-upgrade, N=519739) |
| Proof source | imported from remote DB (`remote_bundle_proof`) | local prover, GPU1, halo2-gpu |

Pre-flight verified the production stack had NOT changed since 8-31 (on-chain `latestVerifier[10]` digest vs newest remote proof instances — exact match), so no fork-point version skew.

## Timeline (UTC)

| Time | Event |
|---|---|
| 06:06 | Phase A up: `10-follow-up.sh --import-proofs --reset` (fork block 25950750; baseline batch 519730). No coordinator/provers by design. |
| 06:08–06:11 | **Phase A acceptance**: bundles 18961–18964 (ends 519732–519735) finalized purely from imported production proofs; `lastFinalizedBatchIndex` → 519735. |
| 06:15 | Relayer stopped to stage a backlog for the parallel window. |
| 07:26 | **t1**: `30-canary-upgrade.sh` — N=519738 (staged bundle 18965 = only old-path specimen), new wrapper `0x3a25…5577` via genuine `updateVerifier`; routing + digest-differ asserted; coordinator + 1 prover started (GPU1). |
| 07:28 | Relayer restarted → **parallel window**: 18965 (end 519737 < N) finalized through the OLD wrapper via `legacyVerifiers` — `cast run` trace: `MVRV.verifyBundleProof(10,519737)` → `0x8082…::verify`, proof instances carry the old digest1. |
| 07:33 | **Rollback drill**: `31-canary-rollback.sh` — routing restored at 519738, boundary cleared to +∞, new stack stopped; relayer/poll-sync untouched. |
| ~08:28 | **Anvil incident #1**: anvil process died silently (state last persisted 08:27; stdout was discarded by the default launch, so no crash reason). |
| 08:36 | Restored by re-launching anvil with the SAME `--state` file: chainId/`lastFinalized=519737`/rolled-back routing all intact. (Note: this state-file relaunch path is NOT what `re-fork.sh` does — re-fork starts a fresh fork; the state relaunch is lighter and preserved legacy routing + finalize history.) |
| 08:37 | **Post-rollback acceptance**: 18966 (end 519738, first batch ≥ N) finalized via the restored OLD wrapper from an imported proof — "as if the upgrade never happened". |
| 08:45 | **Commit-tx incident**: during anvil warm-up after the restore, the relayer's commit tx for batch 519739 hit a client timeout — but the tx HAD landed (next retry: "transaction already imported"). `commit_tx_hash` never written → infinite `ErrorIncorrectBatchHash` commit retries (5,518 log lines). |
| 10:24 | Repaired: on-chain tx found by scanning blocks (`0x9d97a19a…`, nonce 6, blob tx, status 1), `batch.commit_tx_hash` backfilled, relayer restarted (nonce re-init from chain). Spam stopped. |
| 08:40 | **Re-upgrade**: `30-canary-upgrade.sh` again — N=519739, wrapper `0xA7898f4B…c3a` (plonk `0x0b74Ec1f…1797`), all assertions pass. Idempotency incl. the in-place legacy entry confirmed. |
| 08:44–09:27 | 5 chunks of batch 519739 proved locally on the single GPU (incl. 10–15 min first-task witness fetch). |
| ~10:18 | **t2**: 18967 (end 519739, first ≥ N) proven locally and finalized via the NEW wrapper. Digest-verified: local proof instances digest1 = `0x006770fb…` (new guest); trace: `MVRV` → `0xA7898f4B…::verify`. halo2-gpu SNARK `Create EVM proof` **5.98 s**, GPU peak **8.8 GiB**. |
| ~11:00 | **Anvil incident #2**: anvil STALLED (silent — log ends mid-RPC, no panic; `kill -0` and port checks failed, then it self-recovered). During the stall 18968's finalize tx landed (`0x5170aa92…`, nonce 7, status 1) but the receipt was lost → bundle row stuck `rollup_status=1` while on-chain `lastFinalized` had already advanced to 519740. |
| 11:08 | Repaired (same pattern as 08:45): on-chain finalize found, bundle/batch rows backfilled, relayer restarted. Single healthy anvil confirmed (the "restart" attempt exited on port-in-use — the original had self-recovered). |
| 11:1x | **Soak**: 18968 (end 519740) finalize verified via the NEW wrapper; local proof digest1 = `0x006770fb…` (new guest). 0 unfinalized; quarantine holds remote proofs for both ≥N bundles (rollback-safe). |

## Findings / new issues this run

1. **Anvil 1.7.1 instability under shadow-fork load** — one silent death (08:2x) + one silent stall with self-recovery (11:0x), both around state-persistence/heavy-activity moments, no panic in logs (once logging was enabled). The `--state` relaunch recovery path worked perfectly but is **undocumented** (`re-fork.sh` only covers fresh re-forking, which would lose legacyVerifiers routing mid-canary). Recommended: (a) a small anvil watchdog that relaunches from the state file on health-check failure (analogous to the autossh tunnel watchdog), (b) `01-setup-anvil.sh` should log anvil output to a file by default instead of `/dev/null`.
2. **Relayer lacks on-chain reconciliation for send-timeout transactions (hit twice)** — a tx that lands during an RPC stall is retried forever (commit: `ErrorIncorrectBatchHash` spam; finalize: bundle stuck `rs1` while `lastFinalized` on-chain has advanced). Both repaired by backfilling from on-chain evidence. Production-hardening item: before retrying, reconcile against chain state (`committedBatches`, `lastFinalizedBatchIndex`, receipt-by-nonce).
3. **Single-GPU sizing confirmed**: with GPU0 unavailable, 1×4090 (halo2-gpu build) kept pace with mainnet end-to-end (chunks+batch+bundle ≈ 15–40 min per bundle incl. witness fetch vs ~1.5–2 h mainnet cadence).
4. **Staging dependency**: mainnet's low-production morning allowed only ONE staged old-wrapper specimen; the rollback drill supplied the second. Without staging (8-31 run) there may be zero — supports adding `--stage-backlog K` to the runbook.
5. `30-canary-upgrade.sh` WARN "old assets not found … skipping VK-diff sanity check" is expected in canary mode (Phase A has no local assets); the step-f digest-differ check still covers the "is this a real upgrade" question.
6. Confirmed (not fixed this run): `make follow-report` does NOT split at `CANARY_AT_BATCH` (script reads only `UPGRADE_AT_*`) despite `30-canary-upgrade.sh`'s closing hint.

## Acceptance summary

| Criterion | Status |
|---|---|
| Phase A: ≥3 backlog bundles finalized purely from imported production proofs | ✅ (18961–18964, 06:08–06:11) |
| t1: genuine `updateVerifier`, routing asserted both sides of N, digest differ | ✅ (N=519738, wrapper `0x3a25…`) |
| Parallel window: old-proof bundle finalizes post-t1 via legacy routing | ✅ (18965, trace-verified) |
| Rollback drill: old wrapper restored at N; next ≥N bundle finalizes from imported proof | ✅ (18966 via restored old wrapper) |
| Re-upgrade after rollback (idempotent) | ✅ (N=519739, wrapper `0xA7898f4B…`) |
| t2: first ≥N bundle proven locally, finalized via NEW wrapper, digest-verified | ✅ (18967, `0x006770fb…`) |
| Soak: second new-stack bundle through the new wrapper | ✅ (18968, digest-verified) |
| `finalized_lag` back to 0 and quarantine rollback-safe | ✅ (0 unfinalized; 2 proofs ≥ N quarantined) |
| Crash-recovery under fire (unplanned) | ✅ 2× anvil incidents + 2× unrecorded-tx repairs, no test invalidation |

---

## Re-validation run (post-doc-restructure, 2026-09-11 12:49–16:50 UTC)

Full teardown → setup → canary sequence re-run after the trap-corpus restructure, strictly following the new docs — the "does the documentation actually walk?" acceptance.

| Phase | Time (UTC) | Result |
|---|---|---|
| Teardown | 12:49 | stack + `.work` + `remote_bundle_proof` (18 851 rows) + containers |
| Phase A | 12:56–13:10 | 4 bundles (18966–18969) finalized on imported proofs, zero local proving |
| t1 | 13:35 | caught 18970 unproven → **N=519743**; wrapper `0x3a251c0c…5577` via genuine `updateVerifier` |
| Rollback drill | 13:44 | old wrapper restored at N (tx `0x1d38…8c68`); 18970 finalized from quarantined remote proof via OLD wrapper |
| Re-upgrade | 13:48 | N′=519744, wrapper `0xA7898f4B…c3a` (deterministic, same address as the morning run) |
| t2 | 16:45 | 18971 locally proven → finalized via NEW wrapper; `lastFinalizedBatchIndex`=519746, lag=1 |

Zero stack failures, zero unplanned interventions; anvil stable throughout (Trap 44 did not recur). One transient `ErrorIncorrectBatchHash` (Trap 4 selector) during Phase A relayer warm-up, self-healed on the next tx.

Findings:

1. **Fixed — `monitor-catchup.py` wrote metrics to a foreign hardcoded path** (`/home/scroll/zzhang/…` for both `LOG_FILE` and `WORK_DIR`), so `make follow-status` always printed "(no metrics yet)" and the verifier-drift check read a stale `verifier.env`. Now derived from `__file__` (env-overridable via `SHADOW_WORK_DIR`/`SHADOW_METRICS_LOG`); verified end-to-end.
2. GUIDE prereq said "RDS tunnel + autossh" but this box reaches RDS **directly over LAN** (`192.168.1.108:15432`, `local-secrets.md`) — prereq text now covers both routes.
3. Anvil's listen port (`:18545`) and the single-GPU bundle-pipeline latency (~1–1.5 h at rs1/p1 before the bundle row flips) were undocumented; both added to follow/GUIDE.md.

Benign observations: `assets_v2` absent is fine for canary (`30-` warns + skips the VK diff); dropping `remote_bundle_proof` is unnecessary for freshness — poll-sync re-upserts the full remote-prove history (~18 k rows) within minutes.
