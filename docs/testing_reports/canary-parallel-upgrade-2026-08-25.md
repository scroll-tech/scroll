# Canary Parallel-Upgrade Test Report — 2026-08-25

**Mode**: follow-mode canary parallel upgrade (new; extends, does not replace, the `20-upgrade.sh` hard-switch mode).
**Host**: fresh machine (this repo checkout `c500b7d6`), 2×RTX 4090, shadow DB in local docker postgres:15 (`localhost:5433/shadow_rollup`).
**Remote source DB**: `192.168.1.108:15432/mainnet_rollup` (read-only; no local tunnel this time).

## Stacks under test

| | Old (production) | New (this checkout) |
|---|---|---|
| Guest | `79c1f8c` | zkvm-prover `bf887150` (v0.9.0) |
| bundle digest1 | `0x00398b78…9269` | `0x006770fb…5655` |
| Wrapper | `0x808297224e86b1a6055B5F790a2cE07Ed611f955` (mainnet `latestVerifier[10]`, startBatch 0) | deployed fresh on fork per run |
| Proof source | imported from remote DB (`remote_bundle_proof` quarantine) | local provers (GPU) |

Production digests were verified by decoding `bundle.proof` `instances` bytes 384–448 from the remote DB against the on-chain wrapper immutables (exact match). Note production is NOT on the S3 `releases/v0.9.0` guest — AGENTS.md's `0x0dE1…` wrapper note is stale.

## Timeline (UTC)

| Time | Event |
|---|---|
| 06:06 | Phase A up: `10-follow-up.sh --import-proofs --reset` (fork 5h back, block 25828765; lastFinalized@fork 519409). No coordinator/provers. |
| 06:10 | Proof import live: 18,577 production bundle proofs quarantined; 3 applied to the backlog. |
| ~06:12–06:20 | **Phase A acceptance**: bundles 18692/18693/18694 (ends 519410/519412/519415) finalized on the fork purely from imported production proofs, zero local proving. `lastFinalizedBatchIndex` 519409 → 519415. |
| 06:21 | Relayer stopped to stage a backlog for the parallel window. |
| 08:59–09:19 | Import-cursor bug found & fixed (see Findings); backlog bundles 18695/18696 (ends 519417/519418) quarantined + applied. |
| 09:22 | **t1**: `30-canary-upgrade.sh` — boundary N=519419, new wrapper `0x3a25…5577` deployed from S3 v0.9.0 assets and registered via genuine `updateVerifier(10, 519419, …)`; old wrapper moved to `legacyVerifiers[10][0]`. Routing asserted both sides of N; on-chain digest-differ check passed (`00398b78…` vs `006770fb…`). New coordinator+provers started. |
| 09:25–09:35 | **Interleaved finalization (key acceptance)**: relayer restarted; backlog bundles 18695/18696 finalized AFTER the upgrade through the OLD wrapper via `legacyVerifiers` routing (finalize access list includes MVRV + old wrapper; tx `0x73f7dbdb…` status true). Fork `lastFinalizedBatchIndex` → 519418. |
| 09:33 | **Rollback drill**: `31-canary-rollback.sh` — new stack stopped, `updateVerifier(10, 519419, OLD_WRAPPER)` swap-back, boundary cleared. |
| 09:59 | **Post-rollback acceptance**: next mainnet bundle 18697 (end 519420 ≥ N) finalized via imported production proof through the restored old wrapper — "as if the upgrade never happened". Fork `lastFinalizedBatchIndex` → 519420. |
| ~10:02 | **Re-upgrade**: `30-canary-upgrade.sh` again — new boundary N=519421, new wrapper `0xA7898f4BE5Af8A9BCA430E65c9bcbfD34d6b7c3a` registered (plonk verifier `0x0b74Ec1f…1797`), routing + digest checks pass. Rollback → re-upgrade cycle confirmed idempotent. |
| 10:18–12:17 | Mainnet batch-production pause (low activity); batch tip flat at 519421. During the pause both provers died on their first **bundle** task: missing `~/.openvm/params/kzg_bn254_23.srs` (Trap 12 — SRS files were in `~/.openvm/` root, not `params/`). Fixed 14:11, provers restarted 14:12 (see Findings 8). |
| 14:48 | **t2 ✅**: bundles 18698 (end 519421) and 18699 (end 519422) proven by the local NEW stack (`proved_at` 14:48:14 / 14:48:35) and finalized through the NEW wrapper — finalize txs `0x5a2537c7…b0bd9` / `0x60be9f62…4cf39` both status true, access lists include new wrapper `0xA7898f4B…` + plonk verifier `0x0b74Ec1f…`. Fork `lastFinalizedBatchIndex` → 519422. Proof-origin verified cryptographically: bundle 18698's `instances` digest1 = `0x006770fb…5655` (new guest), not production `0x00398b78…9269`. |

## Findings / bugs fixed during the run

1. **Import cursor high-watermark skipped late proofs** (real bug, fixed): mainnet creates the `bundle` row first and attaches the proof minutes later; the importer advanced its cursor to remote `MAX(index)` over ALL bundles, permanently skipping bundles still unproven at scan time. Fixed in `sync-mainnet-db.py`: cursor advances over `proving_status=4` rows only + 200-row lookback re-scan. Self-heals on restart.
2. **Boundary N must exceed the proven backlog** (fixed in `30-canary-upgrade.sh` step b): with a proven-but-unfinalized backlog, `N = lastFinalized+1` would route old-proof bundles to the new wrapper → guaranteed `VerificationFailed`. N is now `max(U, P+1)` with P = max end_batch of unfinalized bundles holding a quarantined proof.
3. **`coordinator_api` needs `conf/genesis.json` relative to CWD** (fresh-machine trap): it CRITs seconds after the startup pid check, provers then exhaust login retries and die. Fixed by copying `tests/prover-e2e/mainnet-galileoV2/genesis.json` into `coordinator/build/bin/conf/`; sanity check added to 30.
4. **`setsid nohup … & echo $!` pidfiles can point at a zombie**: `setsid` forks when the job is a pg leader, so the recorded pid is a dead intermediate; `kill -0` on the zombie succeeds and the real process survives "stops". Hit during the rollback drill (the old api kept port 8390 → next api start failed `bind: address already in use`). `stop_component` in 30/31 now always falls through to the pattern sweep.
5. **Coordinator wasted GPU on already-finalized history**: at t1 the new coordinator began proving chunks of bundles finalized in Phase A (their rows had `proving_status=1` since Phase A never needed local proofs). Mitigated manually with `UPDATE chunk/batch SET proving_status=4 … WHERE bundle finalized` (the same NULL-proof-is-safe pattern as 10-follow-up step b). Candidate improvement: have 30-canary-upgrade.sh run this marking itself.
6. Two stale `prover_task` status-3 rows appeared after the rollback/restart (chunk tasks of an already-finalized batch failed verification post-restart — consistent with the Trap-38 "wipe prover db on restart" lesson). Deleted manually; no impact.
7. Remote `l2_block` range copies are slow on this network (server-side `DataFileRead`, minutes per cycle) — proof import latency ≈ one poll cycle (10–15 min). Back-pressure only, no data loss. Manual one-shot `import_remote_proofs()` works around it when staging a boundary.
8. **halo2 SRS not in `~/.openvm/params/` on the fresh machine** (Trap 12, environment miss during setup): the SRS files had been downloaded to `~/.openvm/` root, not the `params/` subdir the prover reads. Chunk/batch STARK proving is unaffected, so the failure only surfaced when the first **bundle** task reached halo2 keygen — both provers panicked (`Failed to open params file …/params/kzg_bn254_23.srs`) and exited, prover-0 at 11:00, prover-1 at 12:25. Fixed by moving `kzg_bn254_{22,23,24}.srs` into `~/.openvm/params/` and restarting the provers; the sweeper had already reset the interrupted bundle task. Setup checklists should verify `ls ~/.openvm/params/` before starting provers.

## Environment notes (fresh machine)

- NVML driver/library mismatch (kernel 580.159 vs userland 580.173) fixed by `modprobe -r nvidia_uvm nvidia_drm nvidia_modeset nvidia && modprobe nvidia` — no reboot needed (no processes held the devices).
- OpenVM CUDA build ICEs with nvcc 12.4 (`cp_gen_be.c` assertion on `load_sign_extend.cu`); **CUDA 12.8 works**. Also needs `libclang-dev` (bindgen), `python3-pycryptodome`/`requests` (sync scripts).
- Repo convention is shadow DB password `shadow_pass` (docker-compose.yml and all script defaults); recreating the container with that password avoided touching any script.
- `local-secrets.md` corrections: Alchemy mainnet must use the key previously listed under sepolia (`/v2/demo` is dead); mainnet DB is at `192.168.1.108:15432` directly (no stunnel on this host). File updated in place.

## Acceptance summary

| Criterion | Status |
|---|---|
| Phase A: ≥3 backlog bundles finalized purely from imported production proofs | ✅ (18692–18694) |
| t1: genuine `updateVerifier`, old wrapper in `legacyVerifiers`, routing asserted | ✅ (N=519419) |
| Parallel finalization: old-proof bundles finalize post-upgrade via legacy routing | ✅ (18695/18696) |
| Rollback: old wrapper restored, next bundle finalized via imported proof; "no upgrade happened" | ✅ (18697) |
| Re-upgrade after rollback | ✅ (N=519421) |
| t2: first NEW-guest proof finalized on-chain through the new wrapper | ✅ (18698 + 18699, digest-verified) |
