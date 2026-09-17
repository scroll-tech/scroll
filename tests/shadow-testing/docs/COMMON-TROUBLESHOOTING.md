# Common Troubleshooting & Pitfalls (Mode-Independent)

> **Read this file first** before starting any shadow fork or shadow coordinator test.
> This file covers pitfalls that apply regardless of mode. Mode-specific traps live in
> [`../follow/TROUBLESHOOTING.md`](../follow/TROUBLESHOOTING.md) (follow mode) and
> [`../snapshot/TROUBLESHOOTING.md`](../snapshot/TROUBLESHOOTING.md) (snapshot replay mode).
>
> - **Registry & statuses**: [`TROUBLESHOOTING.md`](TROUBLESHOOTING.md) — every trap's number, key symptom, file, and lifecycle status (Active / Fixed / Checklist).
> - **Archive**: [`TRAP-ARCHIVE.md`](TRAP-ARCHIVE.md) — retired traps (fixed upstream/harness or absorbed into the checklists below). Numbers are retired, never reused.
> - This directory contains hard-won knowledge from multiple debugging sessions. Blind experimentation repeats documented mistakes.

## Pre-Flight Ritual (Mandatory)

Before executing a single command:

1. [ ] **Read root `AGENTS.md`** — refresh the trap list.
2. [ ] **Read this file plus the per-mode `TROUBLESHOOTING.md`** (`follow/TROUBLESHOOTING.md` or `snapshot/TROUBLESHOOTING.md`) — check if your planned task matches any documented failure mode. The [trap registry](TROUBLESHOOTING.md) maps symptoms to traps.
3. [ ] **Read the per-mode `GUIDE.md`** (`follow/GUIDE.md` or `snapshot/GUIDE.md`) — verify the specific section matching your task (e.g. "Real Verifier Deployment", "Relayer Finalize on the Fork").
4. [ ] **Verify network** — confirm you are testing **Mainnet** or **Sepolia**, and all configs/ports/RPCs match that network.
5. [ ] **Verify target bundle range** — query the DB to confirm:
   - Bundles exist and have `proving_status = 4` (or will be regenerated)
   - Parent batch of the first target batch exists in the DB
   - All bundle end batches have `committedBatches` entries on Anvil (or will be seeded)

## Mainnet vs Sepolia — Decision Table

| Check | Mainnet | Sepolia | Verification Command |
|-------|---------|---------|---------------------|
| DB port | `5433` or `15432` | `25432` | `psql -h localhost -p <PORT> -c "SELECT version();"` |
| L2 RPC | `l2geth-rpc-proxy.mainnet.aws.scroll.io` | `l2geth-rpc-proxy.sepolia.aws.scroll.io` | `curl -X POST <RPC> -d '{"method":"debug_executionWitness","params":["latest"],"id":1}'` |
| Anvil fork URL | `eth-mainnet.g.alchemy.com` | `eth-sepolia.g.alchemy.com` | `cast block-number --rpc-url <URL>` |
| ScrollChain proxy | `0xa13BAF47339d63B743e7Da8741db5456DAc1E556` | `0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0` | `cast call <ADDR> "lastFinalizedBatchIndex()(uint256)"` |
| MVRV | `0x4CEA3E866e7c57fD75CB0CA3E9F5f1151D4Ead3F` | `0x8A360c7F6fca548507017DdeD732bFe7E078F963` | `cast call <ADDR> "latestVerifier(uint256)" 10` |
| L1MessageQueueV2 | `0x56971da63A3C0205184FEF096E9ddFc7A8C2D18a` | `0xA0673eC0A48aa924f067F1274EcD281A10c5f19F` | `cast call <ADDR> "nextUnfinalizedQueueIndex()(uint256)"` |
| Verifier | Copy from mainnet (`anvil_setCode`) | Check MVRV first; may already match | `cast call <MVRV> "getVerifier(uint256,uint256)" 10 <batchIndex>` |

### Sepolia Operational Differences (beyond addresses/ports)

| Dimension | Mainnet | Sepolia | Failure mode if ignored |
|-----------|---------|---------|------|
| **DB scope** | Imported limited range | Full production snapshot (batches 128080+) | Relayer batch committer floods logs with commit retries |
| **`committedBatches`** | Sparse, but fork block usually covers target batches | Sparse; **every bundle end batch must exist** | Missing entry → `ErrorIncorrectBatchHash(0x2a1c1442)` (Trap 4) |
| **`L1MessageQueueV2` reset** | `nextUnfinalizedQueueIndex = 0` usually sufficient | Set to `MIN(total_l1_messages_popped_before)` of the first target batch; **slot 104** (verify with `forge inspect`) | Wrong slot/value → `ErrorFinalizedIndexTooLarge(0x16465978)` (Trap 5, Trap 20) |
| **Anvil gas estimation** | Same as mainnet | `eth_estimateGas` may fail with fee caps present (`Gas=0`) | Trap 9 (archived — fixed upstream; use a recent relayer build) |
| **Sender balance** | Persisted across restarts | **Resets to 0** after Anvil restart | Re-fund EOAs before each relayer start (Trap 10) |
| **Relayer flags** | Standard | Requires `--config <path>` AND `--min-codec-version 10` | Wrong config or immediate exit (Trap 11) |
| **Blob version** | Usually V0 | Anvil 1.0.0 cannot decode BlobSidecar V1 | Set `fusaka_timestamp: 2000000000` in relayer config |
| **Proofs in DB** | May already be current | Old proofs may be several guest versions behind | Must reset `proving_status = 1` to regenerate (Trap 8, Trap 16) |

## Critical Traps (Do Not Skip)

Traps are grouped by theme; numbers are globally unique and stable. Retired traps (7, 9, 12, 13, 25, 26, 28, 29, 30, 36, 43, 46, 47, 48, 51) live in [`TRAP-ARCHIVE.md`](TRAP-ARCHIVE.md).

### Verifier, Digests & Release Assets

### Trap 1: Wrong Verifier Contract or Wrong Digest Form
- **Symptom**: `VerificationFailed(0x439cc0cd)` even with correct digests.
- **Cause A**: Deployed `ZkEvmVerifierPostEuclid` instead of `ZkEvmVerifierPostFeynman`.
- **Cause B**: Used digests in the wrong form. For v0.9.0 the S3 `digest_1.hex` / `digest_2.hex` files are published in **canonical form**, but if you are re-using an old v0.8.0 workflow that converted from Montgomery form, double-check you are not applying the conversion twice.
- **Cause C**: **MVRV routes the batch to the wrong verifier**. The deployed verifier's digests are correct, but `MultipleVersionRollupVerifier.getVerifier(10, batchIndex)` returns an old verifier with different digests. This happens when re-proving bundles with a new prover (new digests) whose batch indices fall in a range still mapped to a legacy verifier.
- **Cause D**: **The wrapper was copied from mainnet via `anvil_setCode`**. `anvil_setCode` copies runtime bytecode but **preserves the original immutables** (`plonkVerifier`, `verifierDigest1/2`, `protocolVersion`). The copied wrapper assembles its `instances` array from the *mainnet* immutables + `keccak256(protocolVersion || publicInput)`, so locally-generated proofs with different digests are rejected even when the plonk verifier binary, public input hash, and proof are all individually correct. Copying is only valid when your proofs intentionally share mainnet's digests (e.g. re-using imported production proofs).
- **Rule**: For guest v0.9.0 proofs, **always use `PostFeynman`** with digests from `.../releases/v0.9.0/bundle/digest_*.hex`, and **verify MVRV routing** before finalizing.
- **Verification — Digests**: Fetch canonical digests from S3 and deploy with `protocolVersion = 10`:
  ```bash
  BASE_URL="https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/v0.9.0"
  DIGEST1=$(curl -fsSL "${BASE_URL}/bundle/digest_1.hex" | tr -d '[:space:]')
  DIGEST2=$(curl -fsSL "${BASE_URL}/bundle/digest_2.hex" | tr -d '[:space:]')
  ```
  The S3 files are already in canonical form; no conversion or proof extraction is required.
- **Verification — MVRV Routing**: Before finalizing, confirm the verifier returned by MVRV for each target batch matches the verifier whose digests match the proofs:
  ```bash
  for idx in 128069 128070 128071; do
    cast call $MVRV "getVerifier(uint256,uint256)(address)" 10 $idx --rpc-url $ANVIL_RPC
  done
  ```
  If any batch returns the old verifier while proofs use new digests, update MVRV:
  ```bash
  cast rpc anvil_impersonateAccount $OWNER --rpc-url $ANVIL_RPC
  cast send $MVRV \
    "updateVerifier(uint256,uint64,address)" \
    10 $START_BATCH $NEW_VERIFIER \
    --from $OWNER --rpc-url $ANVIL_RPC --unlocked
  ```
- **Diagnosis order when `VerificationFailed` appears** (do NOT jump to blaming the Solidity):
  1. Verify the plonk verifier binary matches the deployed contract's runtime code.
  2. Verify `keccak256(abi.encodePacked(protocolVersion, publicInput))` equals the proof metadata `bundle_pi_hash`.
  3. Verify the wrapper's immutables match the proof's digest words (instances bytes 384–416 / 416–448).
  4. Verify MVRV routing for the exact batch index (Cause C).
  5. Only after 1–4 pass, look at contract logic — and even then, **production code is almost certainly correct**: the wrapper's `sub(0x5a0, i)` loop has finalized thousands of bundles on mainnet; "fixing" its direction inverts the hash-word layout and produces a different, equally failing instances set.
- **Access control**: `finalizeBundlePostEuclidV2` has an `OnlyProver` modifier — on a shadow fork, authorize the finalize sender via `addProver` (or `cast rpc anvil_impersonateAccount <prover>`) before sending.

### Trap 14: Coordinator Verifier Assets vs Prover Circuit S3 Paths (v0.9.0)
- **Symptom**: coordinator asset download 403s or prover cannot find circuit apps.
- **Cause**: Starting with v0.9.0, both verifier assets and circuit apps are released under a unified `scroll-zkvm/releases/v0.9.0/` prefix. Earlier versions split them across `v0.8.0/verifier/` and `scroll-zkvm/galileov2/`.
- **Rule**: For v0.9.0, point both coordinator verifier assets and prover `circuits.galileoV2.base_url` at `…/scroll-zkvm/releases/v0.9.0/`. The expected layout is `{chunk,batch,bundle}/<vk>/` for circuits and `verifier/` for coordinator assets. Prefixes per release: [`CURRENT-STACK.md`](CURRENT-STACK.md).

### Chain, RPC & Fork State

### Trap 2: Anvil Forks Wrong Chain
- **Symptom**: `ScrollChain` proxy has no code, or `eth_chainId` returns `534352`.
- **Cause**: Anvil pointed at Scroll L2 RPC instead of Ethereum L1 RPC.
- **Rule**: Anvil must fork **Ethereum L1** (`chainId=1`). The ScrollChain proxy lives on L1.

### Trap 3: L2 RPC Missing `debug_executionWitness`
- **Symptom**: Coordinator panics at startup or chunks never get assigned.
- **Cause**: Public RPC (`mainnet-rpc.scroll.io`, `sepolia-rpc.scroll.io`) blocks debug methods. Chunks containing L1 messages additionally need `scroll_getL1MessagesInBlock` (most chunks at current mainnet height contain none).
- **Rule**: Use **internal** L2 RPC proxies only.

### Trap 19: `miscData.lastCommittedBatchIndex` Desync → `ErrorBatchNotCommitted` (0x227a699e)

- **Symptom**: `finalizeBundlePostEuclidV2` reverts with custom error `0x227a699e` during `eth_estimateGas`, even though `committedBatches[batchIndex]` on the fork matches the DB hash exactly.
- **Cause**: The deployed (Post-Feynman) ScrollChain rejects finalization when `batchIndex > miscData.lastCommittedBatchIndex`. `committedBatches[i]` being set is **not sufficient** — the bound check uses `miscData.lastCommittedBatchIndex` (slot `0xa1`, lowest 8 bytes). On mainnet `lastCommittedBatchIndex` is usually far ahead of `lastFinalizedBatchIndex`; if setup scripts (or manual patches) set it equal to the rewound `lastFinalizedBatchIndex`, every batch between the two becomes un-finalizable.
- **Diagnosis**:
  ```bash
  cast call $SCROLL_CHAIN "miscData()(uint64,uint64,uint32,bool)" --rpc-url $ANVIL_RPC
  # field 1 = lastCommittedBatchIndex, field 2 = lastFinalizedBatchIndex
  cast call $SCROLL_CHAIN "miscData()(uint64,uint64,uint32,bool)" --rpc-url $FORK_URL --block $FORK_BLOCK
  ```
- **Fix**: Patch slot `0xa1`, keeping the high bytes (reserved + flags + timestamp) intact and only replacing the two index fields:
  ```bash
  # lastCommitted=518665 (0x7ea09, real fork value), lastFinalized=518505 (0x7e969)
  cast rpc anvil_setStorageAt $SCROLL_CHAIN 0xa1 \
    0x0000000000000000000000016a537382000000000007e969000000000007ea09 \
    --rpc-url $ANVIL_RPC
  ```
- **Rule**: `01-setup-anvil.sh` now reads the real `lastCommittedBatchIndex` from fork state by default and preserves timestamp/flags. Only pass `--last-committed` when you deliberately want a different value. Do not set it too high either: `commitBatches` requires `parentBatchHash == committedBatches[lastCommittedBatchIndex]`, so an inflated value breaks future commits on the fork.

### Trap 20: `nextCrossDomainMessageIndex` Too Low → `ErrorFinalizedIndexTooLarge` (0x16465978)

- **Symptom**: After fixing Trap 19, finalization reverts with `execution reverted: \x16FYx` (selector `0x16465978`).
- **Cause**: `_afterFinalizeBatch` calls `L1MessageQueueV2.finalizePoppedCrossDomainMessage(totalL1MessagesPoppedOverall)`, which reverts if the new index exceeds `nextCrossDomainMessageIndex` (slot `0x67`). This happens when a manual patch or stale `--state` file left `nextCrossDomainMessageIndex` below the queue indices your batches pop.
- **Diagnosis**:
  ```bash
  cast call $MQV2 "nextCrossDomainMessageIndex()(uint256)" --rpc-url $ANVIL_RPC
  cast call $MQV2 "nextCrossDomainMessageIndex()(uint256)" --rpc-url $FORK_URL --block $FORK_BLOCK
  # also confirm the rolling hash the proof expects exists:
  cast call $MQV2 "getMessageRollingHash(uint256)(bytes32)" $((TOTAL_L1 - 1)) --rpc-url $ANVIL_RPC
  ```
- **Fix**: Bump slot `0x67` to the real fork value (e.g. 998603 = `0xf3ccb`):
  ```bash
  cast rpc anvil_setStorageAt $MQV2 0x67 0x00000000000000000000000000000000000000000000000000000000000f3ccb --rpc-url $ANVIL_RPC
  ```
  Never lower `nextUnfinalizedQueueIndex` (slot `0x68`) below a bundle's `totalL1MessagesPoppedOverall` — that yields `ErrorFinalizedIndexTooSmall` instead.
- **Rule**: `01-setup-anvil.sh` now defensively bumps slot `0x67` to the fork value when it is lower. Decode the revert selector first — `0x227a699e` and `0x16465978` look similar in relayer logs but have different roots.

### Relayer & Senders

### Trap 10: Sender Balance Lost After Anvil Restart
- **Symptom**: `failed to send transaction, err: Insufficient funds for gas * price + value` even after successful gas estimation.
- **Cause**: `anvil_setBalance` funds do not persist across Anvil restarts (they DO survive a `--state` relaunch — Trap 44 — but not a fresh re-fork).
- **Rule**: After every Anvil restart, verify and re-fund sender EOAs before starting the relayer:
  ```bash
  cast balance 0x410E7FD80a3Fc1E62A4D3450d11b71b812006eB9 --rpc-url http://localhost:18546
  ```

### Trap 11: Relayer Started Without Required Flags
- **Symptom**: Relayer prints help and exits with `Required flag "min-codec-version" not set`, or connects to wrong DB.
- **Cause**: `ROLLUP_RELAYER_CONFIG` env var is NOT supported. The relayer uses `--config` CLI flag.
- **Rule**: Always start relayer with BOTH flags:
  ```bash
  ./rollup_relayer --config /path/to/config.json --min-codec-version 10
  ```

### Prover Environment & Proving Pipeline

### Trap 16: Stale Proofs After Source-Code Revert / Restore

- **Symptom**: Batch proof fails with `VM error: execution error: program exit code 1`, or bundle proof fails with `mismatch batch-proof exe commitment: expected=..., got=...`.
- **Cause**: The shadow DB contains chunk/batch proofs generated by a different code version (e.g. before a `git revert` and subsequent restore of v0.9.0 source adaptations). The stored proofs verify individually because the coordinator loads the same verifier, but the next-level prover rejects their execution commitment.
- **Rule**: After any non-trivial source change (Rust guest code, OpenVM version, `Cargo.lock`, or `rust-toolchain`), **treat existing shadow proofs as suspect**. Reset all relevant chunks, batches, and bundles to `proving_status = 1, proof = NULL` and re-prove from the lowest level. Do not rely on `proving_status = 4` alone.

### Trap 17: `coordinator_cron` Re-Marks Corrupt Bundles as Ready

- **Symptom**: Log is flooded with `format bundle prover task failure: unexpected end of JSON input` for high-index bundles, wasting prover cycles.
- **Cause**: `coordinator_cron` periodically scans batches and sets `bundle.batch_proofs_status = 2` when all member batches have `proving_status = 4`. If those batch proofs are `NULL`/corrupt (Trap 16), the bundle becomes "ready" and is assigned repeatedly.
- **Rule**: While cleaning up stale proofs, either:
  1. Stop `coordinator_cron` until all corrupt proofs are regenerated, or
  2. Reset **all** corrupt bundles/batches/chunks (`proving_status = 4 AND proof IS NULL`) to `proving_status = 1` so the cron never sees them as ready.

### Trap 21: Stale Cached Proof in Prover Local DB → VData Shape Mismatch

- **Symptom**: Coordinator logs `proof generated by prover failed ... Proof shape verification failed: Invalid VData: Proof trace_vdata length (44) does not match number of AIRs (42)` with `proofTime=2` (i.e. instant submission from cache, not a real proof run).
- **Cause**: The prover's local LevelDB (`<prover-workdir>/db`) caches proofs keyed by task. A proof generated under a different circuit/asset version (e.g. before an S3 asset refresh or code revert/restore) is replayed and rejected by the coordinator's shape check.
- **Rule**: Fresh proofs (proofTime ~300s for bundles) from other provers succeed for the same task, so this is self-healing as long as one prover has a clean cache. To stop a prover from repeatedly burning attempts on a poisoned cache, stop it, delete `<prover-workdir>/db`, and restart. When switching circuit versions, wipe all prover local DBs as part of the reset ritual (same spirit as Trap 16).

### Trap 24: `coordinator_cron` Collection Timeouts Shorter Than Real Proof Times → False Timeouts, Duplicate Dispatch, Attempt Exhaustation [both]

- **Symptom**: Repeated `proof task have reach the timeout` warnings in coordinator logs for the **same task id**; the same bundle gets proved twice by different provers; submissions race and the loser gets a benign `validator failure chunk/batch have proved and verified success` reject from `proof_receiver.go`.
- **Cause**: The timeout checker (`coordinator/internal/controller/cron/collect_proof.go`) scans the `prover_task` table every cycle and marks tasks timed out after `{chunk,batch,bundle}_collection_time_sec`. On timeout it invalidates the `prover_task`, decrements `active_attempts`, and **permanently fails the task once `total_attempts >= session_attempts` (5)**. Observed incident: `configs/coordinator.json` had `bundle_collection_time_sec = 180` while real bundle proofs take ~335 s (up to ~90 min for 30-batch bundles). Every bundle task falsely timed out at 180 s, got duplicate-dispatched to a second prover (2× GPU waste), and the two submissions raced.
- **Fix**: Set the collection timeouts well above real proof times in `configs/coordinator.json` (and `configs/coordinator.json.template`):
  - `bundle_collection_time_sec ≥ 7200`
  - `chunk_collection_time_sec = 3600` and `batch_collection_time_sec = 180` are fine vs ~117 s / ~25 s actuals.
  - **Remember**: the timeout checker runs in `coordinator_cron`, so the **cron's** config is the one that matters, not `coordinator_api`'s.
- **Structural gap**: **Note (fixed upstream)**: the coordinator now refunds the charged attempt (both `total_attempts` and `active_attempts`, status back to unassigned) when dispatch fails after `Update*Attempts` — see `recoverAttempts` in `internal/logic/provertask/*_prover_task.go` and `orm.RefundAttemptsByHash` — so format-failure paths no longer leave invisible half-charged tasks; the sweeper reset below is belt-and-braces. Historical description: if task formatting fails *after* attempts are incremented but *before* the `prover_task` row is inserted (e.g. the Trap 22 block-hash failure), no `prover_task` row exists and the timeout checker can never see it. The `sweep-stale-proving.sh` daemon is the safety net — this is why the sweeper also resets `total_attempts`, not just `proving_status`.

### Anvil & On-Chain Reconciliation

### Trap 44: Anvil Silent Death / Stall Mid-Run — `--state` Relaunch Recovery [both]

- **Symptom 1**: all RPC to the Anvil port suddenly refused; the anvil process is gone; the `--state` file's mtime is minutes old (incident 2026-09-11 ~08:28, report `testing_reports/canary-parallel-upgrade-2026-09-11.md`).
- **Symptom 2**: Anvil stops responding (RPC hangs, port/pid checks fail) and then **self-recovers** minutes later; during the stall window the relayer logs `context deadline exceeded` / `connection refused` bursts (incident 2026-09-11 ~11:00).
- **Cause**: Anvil 1.7.1 instability under long-running fork load (periodic mining + blob txs + periodic `--state` persistence + lazy fork-backend fetches). Both incidents were silent — the default launch in `01-setup-anvil.sh` discards stdout/stderr to `/dev/null`, so no crash reason is capturable afterwards.
- **Recovery (validated 2026-09-11)**: relaunch Anvil with the **same command line including `--state <file>`** — it loads the persisted state and resumes exactly where it left off (MVRV legacy routing, `lastFinalizedBatchIndex`, EOA balances/nonces all preserved; block number continues). This is much lighter than `make re-fork` (fresh fork + wrapper redeploy) and is the **only** correct recovery mid-canary/mid-upgrade, where a fresh re-fork would lose `legacyVerifiers` routing and on-chain finalize history. After relaunch, verify `eth_chainId == 1`, `lastFinalizedBatchIndex`, and `getVerifier` routing on both sides of any boundary before restarting the relayer.
- **Watch out**: (a) during warm-up after relaunch Anvil RPC is slow — a relayer send can time out client-side while the tx still lands (→ Trap 45); (b) when relaunching manually, redirect output to a log file (`.work/anvil-restart.log`) instead of `/dev/null`; (c) if the original process only stalled (Symptom 2), a relaunch attempt will exit on port-in-use — check `pgrep -ax anvil` first and reuse the recovered instance.
- **Prevention (candidate)**: an Anvil watchdog (health-check + relaunch from state file), analogous to the `autossh` RDS-tunnel watchdog; plus state-file mtime freshness in the hourly monitor.

### Trap 45: Tx Landed On-Chain but Relayer Never Recorded It → Infinite Retries / Stuck Rows [both]

- **Symptom 1**: commit spam — `Failed to send commitBatch tx ... err="failed to get fee data ... execution reverted: *\x1c\x14B"` (selector `0x2a1c1442`, `ErrorIncorrectBatchHash`) repeating every ~2 s forever; the `batch` row shows `rollup_status = 5` but `commit_tx_hash IS NULL`. Observed 2026-09-11: 5,518 retry lines.
- **Symptom 2**: a `bundle` row stuck at `rollup_status = 1` while on-chain `lastFinalizedBatchIndex` has already advanced past its end batch (the finalize tx landed during an Anvil stall, receipt lost).
- **Cause**: the relayer's tx send hit a **client-side timeout** (slow Anvil — e.g. Trap 44 warm-up) *after* the tx was accepted by the node ("transaction already imported" on the next attempt proves it). The confirmation path never runs, so the DB columns (`commit_tx_hash` / `finalize_tx_hash` + rollup status) are never written; retries then fire against chain state that has already moved on. The relayer performs **no on-chain reconciliation before retrying**.
- **Fix (validated twice, 2026-09-11)** — reconstruct from on-chain evidence, backfill the DB, restart the relayer so senders re-initialize nonces from the chain:
  1. Find the tx: scan recent fork blocks for the sender (`cast rpc eth_getBlockByNumber 0x.. true | jq '.transactions[] | select(.from==…)'`) or read the nonce from the relayer error line.
  2. Verify the receipt (`status 1`) and that the call is the expected one (`commitBatches` blob tx / `finalizeBundlePostEuclidV2`).
  3. Backfill: `UPDATE batch|bundle SET <commit|finalize>_tx_hash = '0x…', rollup_status = 5, finalized_at = now() WHERE … AND <col> IS NULL;` then restart the relayer (`06-run-relayer.sh`) — `initializeNonce = max(db_max+1, chain_pending_nonce)` then picks the correct next nonce.
- **Rule**: whenever the relayer error-logs a revert storm **and** the DB disagrees with on-chain `miscData()` / `lastFinalizedBatchIndex`, trust the chain and reconcile the DB first — restarting into a dirty state keeps the spam going. **Upstream improvement candidate**: reconcile against `committedBatches` / `lastFinalizedBatchIndex` / receipt-by-nonce before retrying a send.

## Step-by-Step Checklist

> Phases below are snapshot-replay-shaped (manual setup); follow mode automates them via `10-follow-up.sh` — use them as a mental model there.

### Phase 0: Environment Validation
- [ ] DB reachable on correct port
- [ ] L2 RPC supports `debug_executionWitness`
- [ ] Anvil not already running on target port
- [ ] Coordinator port 8390 free
- [ ] Prover GPU available (`nvidia-smi`) — including **not occupied by unrelated processes** (high `memory.used` with no prover running; 2026-09-11 run used `GPUS=1` for its entirety because GPU0 held a foreign 22.4 GiB process — 1×4090 suffices for follow/canary pace)
- [ ] halo2 SRS files present: `ls ~/.openvm/params/kzg_bn254_2{2,3,4}.srs` — only bites at the first **bundle** proof, hours in (Trap 12, archived)
- [ ] Toolchain present: `cast`/`forge`/`anvil` (foundry), `psql`, `jq`, `curl`, `go` (coordinator/relayer builds), `cargo` + `nvcc` (GPU prover build)
- [ ] `libclang` installed — `librocksdb-sys` (via scroll-proving-sdk) runs bindgen during the prover build and panics with "Unable to find libclang" without it (`sudo apt install libclang-dev`)
- [ ] Python deps: `psycopg2` AND `pycryptodome` (import name `Crypto`). Note Ubuntu's `python3-pycryptodome` ships the **`Cryptodome`** namespace, which does NOT satisfy `from Crypto.Hash import keccak` in `lib/sync-queue-hashes.py` — install the pycryptodome wheel into the user site (`~/.local/lib/python3.x/site-packages`) instead
- [ ] Fresh shadow DB migrated before first baseline (`db_cli migrate` — Trap 30, archived)

### Phase 1: DB Setup
- [ ] Import bundle range from production RDS
- [ ] **Exclude `proof` columns** from import
- [ ] Reset `proving_status = 1` for chunks, batches, bundles
- [ ] Insert missing parent batch skeleton
- [ ] Populate `l2_block` table and link via `chunk_hash`

### Phase 2: Anvil Fork Setup
- [ ] Start Anvil forked from **Ethereum L1** (not Scroll L2)
- [ ] Verify `eth_chainId == 1`
- [ ] Fund owner and sender accounts (verify balances after any Anvil restart)
- [ ] Add prover EOA to `ScrollChain`
- [ ] Set `lastFinalizedBatchIndex` to `(first_target_batch - 1)`
- [ ] Set `lastCommittedBatchIndex` to mainnet value (do NOT reset to lastFinalized)
- [ ] (Sepolia) Verify end-batch `committedBatches` hashes are non-zero on Anvil
- [ ] (Sepolia) Set `L1MessageQueueV2.nextUnfinalizedQueueIndex` to pre-finalization value:
  ```sql
  SELECT MIN(total_l1_messages_popped_before)
  FROM chunk
  WHERE batch_hash = (SELECT hash FROM batch WHERE index = <first_target_batch>);
  ```
- [ ] (Sepolia) **Verify slot number with `forge inspect L1MessageQueueV2 storage-layout`** before `anvil_setStorageAt`

### Phase 3: Verifier Setup

**Determine which scenario you are in:**

**Scenario A — Re-using production proofs**
- [ ] Extract digests from proof instances
- [ ] Query Sepolia MVRV: `cast call <MVRV> "getVerifier(uint256,uint256)" 10 <batchIndex>`
- [ ] Query verifier digests: `cast call <verifier> "verifierDigest1()"` / `"verifierDigest2()"`
- [ ] If digests match → **skip deployment**, use existing verifier
- [ ] If digests DON'T match → you are actually in Scenario B

**Scenario B — Testing new guest / circuit version**
- [ ] Generate new proofs with the new prover (coordinator + prover pipeline)
- [ ] Extract digests from **newly-generated** proof instances, or fetch them from the S3 release (`digest_*.hex`, canonical for v0.9.0+)
- [ ] Deploy plonk verifier from `coordinator/build/bin/assets_v2/verifier.bin`
- [ ] Deploy `ZkEvmVerifierPostFeynman` with new digests + `protocolVersion = 10`
- [ ] Register on `MultipleVersionRollupVerifier` via `updateVerifier(10, startBatch, verifier)`
- [ ] Verify with `getVerifier(10, batchIndex)`

### Phase 4: Coordinator + Prover
- [ ] Coordinator config points to correct `assets_v2/` directory
- [ ] Coordinator L2 RPC is internal/debug-enabled
- [ ] Prover config `base_url` uses correct S3 path (check [`CURRENT-STACK.md`](CURRENT-STACK.md))
- [ ] Start coordinator, wait for `Start coordinator api successfully`
- [ ] Start prover(s), verify `Got task from coordinator`

### Phase 5: Relayer Finalize
- [ ] Build relayer with latest code (rebuild if `estimategas.go` was patched for Anvil)
- [ ] Relayer config has `dry_run: false`, correct contract addresses
- [ ] Clear stale `pending_transaction` entries
- [ ] Reset target bundles/batches to `rollup_status = 1`
- [ ] **Sync `batch.withdraw_root` from batch proof metadata** (shadow fork only; see snapshot-mode Trap 56)
- [ ] **Start relayer with `--config <path>` AND `--min-codec-version 10`**
- [ ] Monitor logs for `finalizeBundle in layer1` success
- [ ] Verify `lastFinalizedBatchIndex` advanced on Anvil

## When Things Go Wrong (most frequent)

The full symptom→trap mapping lives in the [trap registry](TROUBLESHOOTING.md) (symptom column). The most frequent entries:

| Error / Symptom | Most Likely Cause | See |
|-----------------|-------------------|-----|
| `VerificationFailed(0x439cc0cd)` | Wrong verifier type, digest mismatch, or wrong bundle withdrawRoot (Trap 56); in follow mode: post-fork L1 queue hashes missing (Trap 23); copied mainnet wrapper via `anvil_setCode` (Trap 1 Cause D) | Trap 1, Trap 23, Trap 56 |
| `ErrorIncorrectBatchHash(0x2a1c1442)` | Sparse `committedBatches` (Trap 4), or a commit that already landed unrecorded (Trap 45) | Trap 4, Trap 45 |
| `ErrorFinalizedIndexTooLarge(0x16465978)` | `nextUnfinalizedQueueIndex`/`nextCrossDomainMessageIndex` too low | Trap 5, Trap 20, Trap 23 |
| Follow mode stalls: `failed to fetch block hashes of a chunk` | Poll-synced chunks lack `l2_block` linkage | Trap 22 |
| Chunks verified but batch/bundle sessions never start | NULL `batch_hash`/`bundle_hash` parent links, or stale `prover_task` failure rows | Trap 22, Trap 34 |
| Relayer cannot send on the fork (balance / `ErrorCallerIsNotSequencer` / nonce stuck) | EOA funding, sequencer auth, `pending_transaction` desync | Trap 27 |
| RPC to Anvil refused, or Anvil hung then self-recovered mid-run | Anvil death/stall — relaunch with the same `--state` file (NOT `make re-fork` mid-upgrade) | Trap 44 |
| `Failed to send commitBatch … ErrorIncorrectBatchHash` spam every ~2s, or bundle stuck `rs1` while on-chain `lastFinalized` already advanced | Tx landed on-chain but relayer never recorded it (send timeout) — reconcile DB from chain, restart relayer | Trap 45 |
| RDS bill spikes while a stack runs | Sync/ad-hoc queries missing `deleted_at IS NULL` | Trap 41, [`rds-query-rules.md`](rds-query-rules.md) |
| `mismatched post-state root` | Fork block predates the codec's hardfork | Trap 50 |
| `ErrorCallerIsNotProver (0x7b263b17)` on finalize | Finalize sender not an authorized prover on the fork | Trap 29 (archived) |
| `CoordinatorEmptyProofData` every ~20 s | **Normal idle-poll** when no task is available; only a problem when ALL provers spin AND coordinator logs task-format failures | Trap 22, Trap 21 |

## Historical incidents (moved)

The former "Lessons from the v0.9.0 Multi-Bundle Shadow Test" section (bundles 17297–17301 / 20000–20004, 2026-05–2026-07) has been redistributed:

- **Full postmortem** → [`../../docs/testing_reports/snapshot-early-dryrun-and-relayer-tests.md`](../../docs/testing_reports/snapshot-early-dryrun-and-relayer-tests.md)
- **withdrawRoot from proof metadata** → snapshot-mode Trap 56
- **`anvil_setCode` preserves immutables** → Trap 1 Cause D
- **Stale proofs after code reverts / proof-chain resets** → Trap 16 / Trap 17
- **S3 digest encoding (canonical vs Montgomery)** → [`bundle-digest-encoding.md`](bundle-digest-encoding.md) and [`CURRENT-STACK.md`](CURRENT-STACK.md)
- **Cargo.lock discipline on zkvm pin bumps** → follow-mode Trap 32

Still-unique procedural knowledge from that era, kept here in condensed form:

- **`finalizeBundlePostEuclidV2` is the only finalize selector** (`0xc1aa4e19`); the verifier it calls is chosen by MVRV routing and must be a PostFeynman-style wrapper (`keccak256(protocolVersion || publicInput)`, `protocolVersion = 10` for GalileoV2). Public input layout (204 bytes): `chainId(8) | msgQueueHash(32) | numBatches(4) | prevStateRoot(32) | prevBatchHash(32) | postStateRoot(32) | batchHash(32) | withdrawRoot(32)`.
- **Prover aggregation needs deferral enabled** (OpenVM v2+): batch proving requires the chunk child prover, bundle proving the batch child prover — the prover config must carry `child_circuit_vks` so the right child assets load.
- **Sizing**: single-batch bundle proofs ~10–20 min; 30-batch bundles ~30–90 min per bundle on one GPU.

## Documentation Priority

When debugging, read docs in this order:

1. [Trap registry](TROUBLESHOOTING.md) (symptom → trap) + this file + the per-mode `TROUBLESHOOTING.md` (`follow/` or `snapshot/`) — fastest path to known traps
2. The per-mode `GUIDE.md` (`follow/GUIDE.md` or `snapshot/GUIDE.md`) — detailed setup and procedures
3. `README.md` — quick reference for common commands + documentation conventions
4. [`CURRENT-STACK.md`](CURRENT-STACK.md) — dated version-sensitive facts
5. `../../AGENTS.md` (repo root) — cross-network rules and secrets reference
