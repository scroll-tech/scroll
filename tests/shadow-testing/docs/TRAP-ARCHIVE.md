# Trap Archive

Retired traps: failure modes **fixed upstream or in the harness**, or **absorbed into the setup checklists**. Bodies are kept verbatim for archaeology and for anyone running older builds — knowledge of an old failure is still knowledge.

- Statuses are tracked in the [trap registry](TROUBLESHOOTING.md); a trap lands here only when its number is **retired (never reused)**.
- These traps still `doc-check` as definitions, so `Trap N` references from other documents keep resolving.

## Fixed upstream or in the harness

### Trap 7: Relayer Nonce Desync

> **Status: Fixed** — upstream `chain_nonce_only` sender option + `10-follow-up.sh --reset` TRUNCATEs `pending_transaction`.

- **Symptom**: Tx sent but never mined; `eth_getTransactionReceipt` returns null forever.
- **Cause**: `pending_transaction` table retains nonces from previous runs that were never confirmed. Relayer initializes nonce from `maxDbNonce + 1`, which is ahead of the on-chain nonce.
- **Rule (older builds)**: After any relayer crash or Anvil restart:
  ```sql
  DELETE FROM pending_transaction WHERE sender_address = '<finalize_sender>';
  ```
  Then restart the relayer.

### Trap 9: Anvil `eth_estimateGas` Rejects Fee Caps

> **Status: Fixed** — upstream `rollup/internal/controller/sender/estimategas.go` sends an explicit non-zero gas cap in the estimation `CallMsg`.

- **Symptom**: `failed to get fee data, err: Out of gas: gas required exceeds allowance: 0`.
- **Cause**: Anvil's `eth_estimateGas` fails when `CallMsg` has `GasFeeCap`/`GasTipCap` set but `Gas` is 0 (Go Ethereum client's default).
- **Rule (older builds)**: the correct fix belongs in upstream `estimategas.go` — do not maintain a local patch.

### Trap 13: Prover Docker `--gpus device=N` + Wrong `CUDA_VISIBLE_DEVICES`

> **Status: Fixed** — `04-prover-up.sh --docker` pins `--gpus "device=N"` and deliberately sets **no** `CUDA_VISIBLE_DEVICES` inside the container (a pinned device is visible as index 0, which is already the right GPU).

- **Symptom**: prover container exits (code 139) with `cudaErrorNoDevice: no CUDA-capable device is detected`; only the GPU-0 prover works.
- **Cause**: `--gpus "device=N"` exposes only that GPU and **renumbers it to index 0** inside the container, so setting `CUDA_VISIBLE_DEVICES=N` inside points at a nonexistent device.
- **Rule (manual launches)**: use `--gpus "device=$i"` with `CUDA_VISIBLE_DEVICES=0` (or `--gpus all` with `CUDA_VISIBLE_DEVICES=$i`).

### Trap 25: Orphaned Poll-Sync + Empty Watermark → Full Mainnet History Backfill

> **Status: Fixed** — all three defenses below are in place in the harness; **do not remove any of them**.

- **Symptom**: Shadow DB suddenly fills with millions of mainnet rows (`sync-poll.log` shows `chunk: inserted 25000/6642569` and counting).
- **Cause**: an orphaned sync daemon (pidfile-invisible) surviving a reset, combined with watermark = 0 on the truncated tables, makes the poll treat all of mainnet history as new rows.
- **Defenses**: (1) `11-follow-stop.sh` kills the whole process group; (2) `MAX_POLL_DELTA = 5000` guard in poll mode; (3) baseline runs before any daemon starts.
- **Recovery**: kill the sync process, then `DELETE` contaminated rows below the baseline window.

### Trap 26: Relayer L2 Watcher Genesis-Crawls an Empty `l2_block`

> **Status: Fixed** — relayer supports `l2_config.disable_l2_watcher` (set by the shadow relayer template); `--reset` also stops all writers before truncating.

- **Symptom**: `l2_block` fills with rows numbered 1, 2, 3, … at ~30–60 blocks/s with NULL `chunk_hash` — a days-long full-history crawl.
- **Cause**: the relayer's L2 watcher loop read `MAX(l2_block.number)` once as 0 (empty table) and started fetching from block 1.
- **Rule**: never let `l2_block` go empty while a relayer is running.

### Trap 28: Post-Fusaka Fork + Anvil 1.0.0 → Blob Base Fee Explosion

> **Status: Fixed** — `01-setup-anvil.sh` mines empty blocks after `wait_for_anvil` until `cast blob-base-fee` < 1 gwei.

- **Symptom**: every `commitBatches` right after a fresh fork fails `estimateGasLimit failure ... Insufficient funds` despite a funded EOA (`cast blob-base-fee` ≈ 1e18 wei).
- **Cause**: pre-Fusaka Anvil prices the inherited `excessBlobGas` with the Dencun update fraction → astronomical blob base fee.
- **Manual equivalent (older harness)**: `cast rpc anvil_mine 400`.

### Trap 29: Same-Block Ordering — Manual `addProver` Mined After the Failing Finalize

> **Status: Fixed** — `10-follow-up.sh` step h re-ensures `isProver(finalize_sender)` idempotently on every run. Recovery knowledge kept for older setups.

- **Symptom**: a finalize tx reverts `ErrorCallerIsNotProver` (0x7b263b17) although `addProver` was sent first; bundle ends at `rollup_status = 7`, which `ProcessPendingBundles` **never retries**.
- **Cause**: Anvil mined both txs in one block, ordering the finalize (txIndex 0) before the `addProver` (txIndex 1).
- **Recovery**: impersonate the owner, `addProver(<finalize_eoa>)`, verify `isProver` = true, then `UPDATE bundle SET rollup_status = 1 WHERE index = <n>`. **Rule of thumb**: a bundle at `rollup_status = 7` is stranded by design and always needs the manual reset after fixing the cause.

### Trap 36: `cast receipt <tx> status` Output Format Changed in Foundry ≥ 1.6

> **Status: Fixed** — the deploy/upgrade scripts accept `0x1`, `1`, `1 (success)`, and `true`.

- **Symptom**: scripts abort with "updateVerifier reverted (… status 'true')" even though the tx succeeded.
- **Cause**: cast output drift across versions (`0x1` → `1 (success)` → `true`); literal string comparison misreads success as failure.
- **Rule**: when in doubt, verify on-chain state instead of trusting the script's verdict.

### Trap 43: `11-follow-stop.sh` Aborts Midway — sourced lib re-enables `set -e`

> **Status: Fixed** (2026-09-04) — `set +e` immediately after the `source`; prover loop matches on `prover.json`; trailing `exit 0`.

- **Symptom**: `make follow-stop` stops some components then exits non-zero, leaving docker provers/coordinators/Anvil running.
- **Cause**: `lib/anvil-utils.sh`'s `set -euo pipefail` silently re-enabled `-e` inside the deliberately `-e`-less stop script.

### Trap 46: `cast code` Returns `0x` for Codeless Accounts — Non-Empty Checks Are Not Enough

> **Status: Fixed** — `01-setup-anvil.sh` explicitly rejects `"0x"` before copying and, in `--skip-verifier` mode, leaves the MVRV untouched; `10-follow-up.sh` asserts the routed wrapper has code and the expected `protocolVersion`.
> Renumbered 2026-09-11 from "Trap 32", which collided with follow-mode's Trap 32 (revm drift).

- **Symptom** (historical): a "copied" verifier address with **no code** on the fork; every finalization would have failed `VerificationFailed`.
- **Cause**: `cast code` prints `0x` for an EOA / nonexistent contract, so `[[ -n "$code" ]]` passes and `anvil_setCode` writes an empty blob.
- **Rule**: after any `cast code`, check non-empty AND non-`0x`; after any registration, verify `cast code <addr>` is non-trivial.

### Trap 47: `set -euo pipefail` + Failing `cast`/`psql` Inside `$( )` → Silent Script Death

> **Status: Fixed** — every `$( )` around `cast`/`psql` in `lib/` and `follow/scripts/` now appends `|| true` and validates explicitly. Renumbered 2026-09-11 from "Trap 33", which collided with follow-mode's Trap 33 (`agg_vk.bin`).

- **Symptom**: an orchestration script stops mid-step with NO error line — `make` reports `Error 1` after the last step header.
- **Cause**: a failing command substitution under `pipefail` + `set -e` exits before the validation/`log_error` below it.
- **Rule**: follow the same pattern when adding new probes — and when a script dies silently, suspect this first.

### Trap 48: `coordinator_api` Reads `conf/genesis.json` Relative to Its CWD

> **Status: Fixed** — `30-canary-upgrade.sh` sanity-checks the file up front; also documented as follow-mode Trap 40 Cause 2. Renumbered 2026-09-11 from "Trap 34", which collided with follow-mode's Trap 34 (stale `prover_task` rows).

- **Symptom**: `coordinator_api` CRITs seconds after start: `failed to read genesis ... open conf/genesis.json`; provers then exhaust login retries and die.
- **Fix**: `cp tests/prover-e2e/mainnet-galileoV2/genesis.json coordinator/build/bin/conf/genesis.json`.

### Trap 51: Multiple Provers Sharing One Circuit-Cache Directory

> **Status: Fixed** — `04-prover-up.sh` gives each GPU its own work dir (`.work/prover-N/`). Kept for manual multi-prover launches.

- **Symptom**: prover dies with `File exists (os error 17)` when two instances write the same temp file under a shared `.work/galileo` cache.
- **Rule (manual)**: one work directory per prover; optionally symlink a shared **read-only** circuit cache into each and keep distinct write dirs.

## Absorbed into setup checklists

### Trap 12: halo2 SRS Not in `~/.openvm/params/`

> **Status: Checklist** — a Phase 0 pre-flight item (COMMON-TROUBLESHOOTING.md); the failure only bites once per fresh machine, hours into a run.

- **Symptom**: chunk/batch proofs succeed; the **first bundle proof** crashes the prover with `Params file ".../.openvm/params/kzg_bn254_23.srs" does not exist`.
- **Cause**: openvm reads the KZG SRS from `$HOME/.openvm/params/kzg_bn254_{22,23,24}.srs` only at the bundle proof's halo2 stage.
- **Check**: `ls ~/.openvm/params/` before starting provers; in docker mode the params dir is mounted read-only at the same path.

### Trap 30: Fresh Shadow DB — Baseline Sync Fails, Schema Never Migrated

> **Status: Checklist** — a Phase 0 pre-flight item ("Fresh shadow DB migrated before first baseline").

- **Symptom**: first-ever `10-follow-up.sh` dies in baseline: `psql staging prep failed ... relation "public.<table>" does not exist`.
- **Cause**: `copy_table_pipe()` needs the real tables to exist; nothing migrates the schema before the coordinator starts.
- **Check**: `cd database && go build -o /tmp/db_cli ./cmd && /tmp/db_cli migrate --config <config-with-shadow-dsn>` once per fresh DB.
