# Common Troubleshooting & Pitfalls (Mode-Independent)

> **Read this file first** before starting any shadow fork or shadow coordinator test.
> This file covers pitfalls that apply regardless of mode. Mode-specific traps live in
> [`../follow/TROUBLESHOOTING.md`](../follow/TROUBLESHOOTING.md) (follow mode) and
> [`../snapshot/TROUBLESHOOTING.md`](../snapshot/TROUBLESHOOTING.md) (snapshot replay mode).
> Trap numbers are stable across all three files — "Trap 22" refers to the same trap everywhere.
> This directory contains hard-won knowledge from multiple debugging sessions. Blind experimentation will repeat documented mistakes.

## Pre-Flight Ritual (Mandatory)

Before executing a single command:

1. [ ] **Read root `AGENTS.md`** — refresh the trap list.
2. [ ] **Read this file plus the per-mode `TROUBLESHOOTING.md`** (`follow/TROUBLESHOOTING.md` or `snapshot/TROUBLESHOOTING.md`) — check if your planned task matches any documented failure mode.
3. [ ] **Read the per-mode `GUIDE.md`** (`follow/GUIDE.md` or `snapshot/GUIDE.md`) — verify the specific section matching your task (e.g., "Real Verifier Deployment", "Multi-Bundle Relayer Finalize Test").
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

## Critical Traps (Do Not Skip)

### Trap 1: Wrong Verifier Contract or Wrong Digest Form
- **Symptom**: `VerificationFailed(0x439cc0cd)` even with correct digests.
- **Cause A**: Deployed `ZkEvmVerifierPostEuclid` instead of `ZkEvmVerifierPostFeynman`.
- **Cause B**: Used digests in the wrong form. For v0.9.0 the S3 `digest_1.hex` / `digest_2.hex` files are published in **canonical form**, but if you are re-using an old v0.8.0 workflow that converted from Montgomery form, double-check you are not applying the conversion twice.
- **Cause C**: **MVRV routes the batch to the wrong verifier**. The deployed verifier's digests are correct, but `MultipleVersionRollupVerifier.getVerifier(10, batchIndex)` returns an old verifier with different digests. This happens when re-proving bundles with a new prover (new digests) whose batch indices fall in a range still mapped to a legacy verifier.
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

### Trap 2: Anvil Forks Wrong Chain
- **Symptom**: `ScrollChain` proxy has no code, or `eth_chainId` returns `534352`.
- **Cause**: Anvil pointed at Scroll L2 RPC instead of Ethereum L1 RPC.
- **Rule**: Anvil must fork **Ethereum L1** (`chainId=1`). The ScrollChain proxy lives on L1.

### Trap 3: L2 RPC Missing `debug_executionWitness`
- **Symptom**: Coordinator panics at startup or chunks never get assigned.
- **Cause**: Public RPC (`mainnet-rpc.scroll.io`, `sepolia-rpc.scroll.io`) blocks debug methods.
- **Rule**: Use **internal** L2 RPC proxies only.

### Trap 7: Relayer Nonce Desync
- **Symptom**: Tx sent but never mined; `eth_getTransactionReceipt` returns null forever.
- **Cause**: `pending_transaction` table retains nonces from previous runs that were never confirmed. Relayer initializes nonce from `maxDbNonce + 1`, which is ahead of the on-chain nonce.
- **Rule**: After any relayer crash or Anvil restart:
  ```sql
  DELETE FROM pending_transaction WHERE sender_address = '<finalize_sender>';
  ```
  Then restart the relayer.

### Trap 9: Anvil `eth_estimateGas` Rejects Fee Caps
- **Symptom**: `failed to get fee data, err: Out of gas: gas required exceeds allowance: 0`.
- **Cause**: Anvil's `eth_estimateGas` fails when `CallMsg` has `GasFeeCap`/`GasTipCap` set but `Gas` is 0 (Go Ethereum client's default).
- **Rule**: If you hit this on a shadow fork, the correct fix belongs in the upstream `rollup/internal/controller/sender/estimategas.go` (do not maintain a local patch in this branch). Verify with the latest `develop` code and, if still present, fix it there so all shadow tests benefit.

### Trap 10: Sender Balance Lost After Anvil Restart
- **Symptom**: `failed to send transaction, err: Insufficient funds for gas * price + value` even after successful gas estimation.
- **Cause**: `anvil_setBalance` funds do not persist across Anvil restarts.
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

### Trap 12: halo2 SRS Not in `~/.openvm/params/`
- **Symptom**: chunk/batch proofs succeed; the **first bundle proof** crashes the prover with
  `Params file ".../.openvm/params/kzg_bn254_23.srs" does not exist`. Bundle stuck at `proving_status=2`.
- **Cause**: openvm reads the KZG SRS from `$HOME/.openvm/params/kzg_bn254_{22,23,24}.srs` only at the
  bundle proof's halo2 stage; if the `.srs` files sit in `~/.openvm/` root (or anywhere else) they are
  silently not found.
- **Rule**: `mkdir -p ~/.openvm/params && mv ~/.openvm/kzg_bn254_2{2,3,4}.srs ~/.openvm/params/`. Mount the
  host openvm dir to `/root/.openvm` (writable) for the prover container and confirm the path resolves.

### Trap 13: Prover Docker `--gpus device=N` + Wrong `CUDA_VISIBLE_DEVICES`
- **Symptom**: prover container exits (code 139) with `cudaErrorNoDevice: no CUDA-capable device is detected`;
  only the GPU-0 prover works.
- **Cause**: `--gpus "device=N"` exposes only that GPU and **renumbers it to index 0** inside the container,
  so `CUDA_VISIBLE_DEVICES=N` points at a nonexistent device.
- **Rule**: use `--gpus "device=$i"` with `CUDA_VISIBLE_DEVICES=0` (or `--gpus all` with `CUDA_VISIBLE_DEVICES=$i`).

### Trap 14: Coordinator Verifier Assets vs Prover Circuit S3 Paths (v0.9.0)
- **Symptom**: coordinator asset download 403s or prover cannot find circuit apps.
- **Cause**: Starting with v0.9.0, both verifier assets and circuit apps are released under a unified `scroll-zkvm/releases/v0.9.0/` prefix. Earlier versions split them across `v0.8.0/verifier/` and `scroll-zkvm/galileov2/`.
- **Rule**: For v0.9.0, point both coordinator verifier assets and prover `circuits.galileoV2.base_url` at `…/scroll-zkvm/releases/v0.9.0/`. The expected layout is `{chunk,batch,bundle}/<vk>/` for circuits and `verifier/` for coordinator assets.

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

### Trap 21: Stale Cached Proof in Prover Local DB → VData Shape Mismatch

- **Symptom**: Coordinator logs `proof generated by prover failed ... Proof shape verification failed: Invalid VData: Proof trace_vdata length (44) does not match number of AIRs (42)` with `proofTime=2` (i.e. instant submission from cache, not a real proof run).
- **Cause**: The prover's local LevelDB (`<prover-workdir>/db`) caches proofs keyed by task. A proof generated under a different circuit/asset version (e.g. before an S3 asset refresh or code revert/restore) is replayed and rejected by the coordinator's shape check.
- **Rule**: Fresh proofs (proofTime ~300s for bundles) from other provers succeed for the same task, so this is self-healing as long as one prover has a clean cache. To stop a prover from repeatedly burning attempts on a poisoned cache, stop it, delete `<prover-workdir>/db`, and restart. When switching circuit versions, wipe all prover local DBs as part of the reset ritual (same spirit as Trap 16).

### Trap 24: `coordinator_cron` Collection Timeouts Shorter Than Real Proof Times → False Timeouts, Duplicate Dispatch, Attempt Exhaustion [both]

- **Symptom**: Repeated `proof task have reach the timeout` warnings in coordinator logs for the **same task id**; the same bundle gets proved twice by different provers; submissions race and the loser gets a benign `validator failure chunk/batch have proved and verified success` reject from `proof_receiver.go`.
- **Cause**: The timeout checker (`coordinator/internal/controller/cron/collect_proof.go`) scans the `prover_task` table every cycle and marks tasks timed out after `{chunk,batch,bundle}_collection_time_sec`. On timeout it invalidates the `prover_task`, decrements `active_attempts`, and **permanently fails the task once `total_attempts >= session_attempts` (5)**. Observed incident: `configs/coordinator.json` had `bundle_collection_time_sec = 180` while real bundle proofs take ~335 s (up to ~90 min for 30-batch bundles). Every bundle task falsely timed out at 180 s, got duplicate-dispatched to a second prover (2× GPU waste), and the two submissions raced.
- **Fix**: Set the collection timeouts well above real proof times in `configs/coordinator.json` (and `configs/coordinator.json.template`):
  - `bundle_collection_time_sec ≥ 7200`
  - `chunk_collection_time_sec = 3600` and `batch_collection_time_sec = 180` are fine vs ~117 s / ~25 s actuals.
  - **Remember**: the timeout checker runs in `coordinator_cron`, so the **cron's** config is the one that matters, not `coordinator_api`'s.
- **Structural gap**: **Note (fixed upstream)**: the coordinator now refunds the charged attempt (both `total_attempts` and `active_attempts`, status back to unassigned) when dispatch fails after `Update*Attempts` — see `recoverAttempts` in `internal/logic/provertask/*_prover_task.go` and `orm.RefundAttemptsByHash` — so format-failure paths no longer leave invisible half-charged tasks; the sweeper reset below is belt-and-braces. Historical description: if task formatting fails *after* attempts are incremented but *before* the `prover_task` row is inserted (e.g. the Trap 22 block-hash failure), no `prover_task` row exists and the timeout checker can never see it. The `sweep-stale-proving.sh` daemon is the safety net — this is why the sweeper also resets `total_attempts`, not just `proving_status`.

## Step-by-Step Checklist

### Phase 0: Environment Validation
- [ ] DB reachable on correct port
- [ ] L2 RPC supports `debug_executionWitness`
- [ ] Anvil not already running on target port
- [ ] Coordinator port 8390 free
- [ ] Prover GPU available (`nvidia-smi`)

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

**Scenario A — Re-using production proofs (like bundles 13445-13449)**
- [ ] Extract digests from proof instances
- [ ] Query Sepolia MVRV: `cast call <MVRV> "getVerifier(uint256,uint256)" 10 <batchIndex>`
- [ ] Query verifier digests: `cast call <verifier> "verifierDigest1()"` / `"verifierDigest2()"`
- [ ] If digests match → **skip deployment**, use existing verifier
- [ ] If digests DON'T match → you are actually in Scenario B

**Scenario B — Testing new guest / circuit version (0.8.0 / openvm 1.6+)**
- [ ] Generate new proofs with the new prover (coordinator + prover pipeline)
- [ ] Extract digests from **newly-generated** proof instances
- [ ] Deploy plonk verifier from `coordinator/build/bin/assets_v2/verifier.bin`
- [ ] Deploy `ZkEvmVerifierPostFeynman` with new digests + `protocolVersion = 10`
- [ ] Register on `MultipleVersionRollupVerifier` via `updateVerifier(10, startBatch, verifier)`
- [ ] Verify with `getVerifier(10, batchIndex)`

### Phase 4: Coordinator + Prover
- [ ] Coordinator config points to correct `assets_v2/` directory
- [ ] Coordinator L2 RPC is internal/debug-enabled
- [ ] Prover config `base_url` uses correct S3 path (no `/releases/` for v0.8.0)
- [ ] Start coordinator, wait for `Start coordinator api successfully`
- [ ] Start prover(s), verify `Got task from coordinator`

### Phase 5: Relayer Finalize
- [ ] Build relayer with latest code (rebuild if `estimategas.go` was patched for Anvil)
- [ ] Relayer config has `dry_run: false`, correct contract addresses
- [ ] Clear stale `pending_transaction` entries
- [ ] Reset target bundles/batches to `rollup_status = 1`
- [ ] **Sync `batch.withdraw_root` from batch proof metadata** (shadow fork only; see Trap 10)
- [ ] **Start relayer with `--config <path>` AND `--min-codec-version 10`**
- [ ] Monitor logs for `finalizeBundle in layer1` success
- [ ] Verify `lastFinalizedBatchIndex` advanced on Anvil

## When Things Go Wrong

| Error / Symptom | Most Likely Cause | See |
|-----------------|-------------------|-----|
| `VerificationFailed(0x439cc0cd)` | Wrong verifier type, digest mismatch, **or wrong bundle withdrawRoot**; in follow mode: post-fork L1 queue hashes missing | Trap 1, **Trap 10**, Trap 23 |
| `ErrorIncorrectBatchHash(0x2a1c1442)` | Sparse `committedBatches`, end batch hash is zero | Trap 4 |
| `ErrorFinalizedIndexTooLarge(0x16465978)` | `nextUnfinalizedQueueIndex`/`nextCrossDomainMessageIndex` too low | Trap 5, Trap 20, Trap 23 |
| Follow mode stalls: `failed to fetch block hashes of a chunk` | Poll-synced chunks lack `l2_block` linkage | Trap 22 |
| Old pending chunks never get sessions, no ERROR logged | `total_attempts >= 5` starvation; sweep must reset `total_attempts` | Trap 22 |
| Chunks verified but batch/bundle sessions never start | NULL `batch_hash`/`bundle_hash` parent links from poll sync | Trap 22 |
| Repeated `proof task have reach the timeout` for the same task id; duplicate proving; `have proved and verified success` rejects | Collection timeout misconfiguration (`*_collection_time_sec` < real proof times) in the **cron's** config | Trap 24 |
| `record not found` (parent batch) | Parent batch not imported | Trap 6 |
| `Out of gas: gas required exceeds allowance: 0` | Anvil gas estimation bug with fee caps | Trap 9 |
| `Insufficient funds for gas * price + value` | Sender balance is 0 on Anvil | Trap 10 |
| Tx sent but never mined | Nonce desync (`pending_transaction` stale) | Trap 7 |
| Relayer exits with `Required flag "min-codec-version" not set` | Missing CLI flags | Trap 11 |
| Coordinator assigns but prover gets nothing | L2 RPC missing `debug_executionWitness` | README.md |
| `CoordinatorEmptyProofData` | Prover crashed; reset stuck tasks | README.md |
| `Params file ".../kzg_bn254_23.srs" does not exist` (bundle proof crash) | halo2 SRS not in `~/.openvm/params/` | Trap 12 |
| Prover exits 139 `cudaErrorNoDevice` | `--gpus device=N` + wrong `CUDA_VISIBLE_DEVICES` | Trap 13 |
| Coordinator asset download 403 (`galileov2/verifier/...`) | Wrong S3 prefix; use `v0.8.0/verifier/` | Trap 14 |
| `l2_block` export hangs for minutes | Slow `chunk_hash` JOIN; export by block-number range | Trap 15 |
| `mismatch batch-proof exe commitment` or batch proof `VM error: execution error: program exit code 1` | Stale chunk/batch proofs generated by an earlier/reverted code version | Trap 16 |
| `format bundle prover task failure: unexpected end of JSON input` in a loop | `coordinator_cron` re-marks corrupt bundles as ready; stop cron or reset all stale proofs | Trap 17 |
| Provers busy but target bundles never prove | Newer bundles/batches created by relayer compete for prover slots | Trap 18 |

## Lessons from the v0.9.0 Multi-Bundle Shadow Test

The following issues were hit while finalizing bundles 17297–17301 (batches 517761–517765) on an Anvil mainnet fork with zkvm guest prover v0.9.0. Keep them in mind for future upgrades.

### 1. Do not `git checkout --` uncommitted source changes blindly

When cleaning up the branch, the v0.9.0 source adaptations (`libzkp`, `prover-bin`, Go `message` types, `rust-toolchain`, `Cargo.lock`) were accidentally reverted because they were not committed. They had to be reconstructed from compiler errors. Always check `git diff --stat` before a bulk revert, and stage or stash anything you intend to keep.

### 2. `finalizeBundlePostEuclidV2` is the only finalize function, but the verifier must be Post-Feynman

`ScrollChain` exposes only one bundle-finalize selector (`0xc1aa4e19`). The name says `PostEuclidV2`, but the verifier it actually calls is chosen by `MultipleVersionRollupVerifier.getVerifier(10, batchIndex)`. For GalileoV2 / v0.9.0 this must be a `ZkEvmVerifierPostFeynman`-style wrapper whose `protocolVersion` immutable is `10`.

- `ZkEvmVerifierPostEuclid` computes `keccak256(publicInput)` — this is old code and will reject current proofs.
- `ZkEvmVerifierPostFeynman` computes `keccak256(protocolVersion || publicInput)` — this matches v0.9.0 `bundle_pi_hash`.

For v0.9.0 GalileoV2 bundle proofs, `publicInput` is encoded as:

```text
| layer2ChainId | messageQueueHash | numBatches | prevStateRoot | prevBatchHash | postStateRoot | batchHash | withdrawRoot |
|    8 bytes    |     32 bytes     |  4 bytes   |   32 bytes    |   32 bytes    |   32 bytes    | 32 bytes  |   32 bytes   |
```

The `messageQueueHash` is included in the public input (derived from `L1MessageQueueV2` by the contract). Older wrapper comments/documents may show the format without `messageQueueHash`; that format is for pre-GalileoV2 proofs.

### 3. Copying the mainnet verifier via `anvil_setCode` fails for new guest versions

`anvil_setCode` copies runtime bytecode but **preserves the original immutables** (`plonkVerifier`, `verifierDigest1/2`, `protocolVersion`). If your local v0.9.0 proofs use different digests than mainnet, the wrapper will return `VerificationFailed`. For a new guest version, deploy a fresh `ZkEvmVerifierPostFeynman` using the S3 release digests.

### 4. MVRV routing must be verified per target batch

After deploying a new verifier, confirm that `MVRV.getVerifier(10, batchIndex)` returns your wrapper for every batch you intend to finalize. If the fork block already contains a later mainnet verifier registration, a plain `updateVerifier` may be rejected; force the storage slot or impersonate the owner as needed for the shadow fork.

### 5. v0.9.0 dependency graph needs a fresh `Cargo.lock`

Pointing `Cargo.toml` to v0.9.0 is not enough. The first `cargo check` hit a revm version conflict because the old `Cargo.lock` pinned incompatible crate versions. Regenerating `Cargo.lock` resolved it.

### 6. Prover aggregation circuits need deferral enabled

OpenVM v2+ requires `Prover::enable_deferral(child_prover)` before proving aggregation tasks:

- Batch proving needs a chunk child prover.
- Bundle proving needs a batch child prover.

The prover config therefore needs `child_circuit_vks` so the prover can load the correct child circuit assets.

### 7. S3 digest files are canonical — no proof extraction needed

For v0.9.0, `.../releases/v0.9.0/bundle/digest_1.hex` and `digest_2.hex` are published in the canonical form expected by the Plonk verifier. Do not apply Montgomery→canonical conversion and do not extract digests from proof `instances` unless you are double-checking a specific artifact.

### 8. Resetting bundles may require regenerating the entire proof chain

When re-testing bundles 20000–20004 after the branch's source adaptations were reverted and restored, the existing chunk and batch proofs in the shadow DB no longer matched the current prover's execution commitments. The only reliable fix was to reset **all** chunks, batches, and bundles in the target range (`proving_status = 1, proof = NULL`) and let the provers rebuild the chain from scratch.

- Do not assume `proving_status = 4` means the proof is compatible with the current code.
- Stop `coordinator_cron` while cleaning stale proofs, or it will re-mark corrupt bundles as ready and flood the log with `format bundle prover task failure`.
- Stop the relayer if you only need a fixed imported range; otherwise live L2 blocks create new batches that compete for prover time.

### 10. Relayer must use the bundle withdrawRoot from proof metadata

**Symptom**: `VerificationFailed(0x439cc0cd)` during relayer finalization even though the verifier digests and MVRV routing are correct.

**Root cause**: The relayer was passing `dbBatch.WithdrawRoot` to `finalizeBundlePostEuclidV2`. In shadow-test setups, the batch table's top-level `withdraw_root` column can be stale:

- `fetch-l2-blocks.py` cannot read `withdraw_root` from a standard L2 RPC, so it writes `0x0...0` into `l2_block.withdraw_root`.
- That placeholder propagates to `chunk.withdraw_root` and `batch.withdraw_root`.
- The prover correctly recomputes the real withdraw root while generating batch proofs and stores it in `batch.proof -> metadata.batch_info.withdraw_root` (and later in the bundle proof metadata).
- The contract uses the passed `withdrawRoot` when computing the public input hash, so a stale `0x0...0` produces a hash that does not match the proof's `bundle_pi_hash`.

**Fix (relayer)**: Use the withdraw root from the verified bundle proof metadata when packing the finalize calldata:

```go
if aggProof.MetaData.BundleInfo == nil {
    return nil, fmt.Errorf("bundle %d proof metadata missing BundleInfo", dbBatch.Index)
}
withdrawRoot := aggProof.MetaData.BundleInfo.WithdrawRoot
```

**Fix (shadow DB)**: After batch proofs are verified, sync the top-level `batch.withdraw_root` column from the proof metadata so the DB is consistent:

```bash
cd tests/shadow-testing/scripts
DB_DSN="postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup" \
    python3 09-sync-batch-withdraw-roots.py --batch-range 517766:517795
```

**Verification**: Decode the relayer's finalize calldata and confirm `withdrawRoot` equals `proof.metadata.bundle_info.withdraw_root`:

```bash
# The 4th argument of finalizeBundlePostEuclidV2(bytes,uint256,bytes32,bytes32,bytes)
python3 -c "from eth_abi import decode; d=bytes.fromhex(open('/tmp/calldata.hex').read().strip()[8:]); \
  print('withdrawRoot:', '0x'+decode(['bytes','uint256','bytes32','bytes32','bytes'], d)[3].hex())"
```

**Production note**: On mainnet the `l2_block.withdraw_root` column is populated correctly, so `batch.withdraw_root` is trustworthy. The relayer change is defensive; the DB sync script is only needed for shadow forks that rely on `fetch-l2-blocks.py`.

### 11. Large bundles (30 batches) are much slower than single-batch bundles

The first successful shadow test finalized five single-batch bundles (17297–17301). The follow-up test targeted bundles 20000–20004, which are mostly 30-batch bundles. Plan timing accordingly:

- Single-batch bundle proof: ~10–20 minutes.
- 30-batch bundle proof: ~30–90 minutes, depending on block complexity and GPU.

With four GPUs, five 30-batch bundles can take 1–3 hours for the bundle proofs alone, after chunk and batch proofs are ready.

## Documentation Priority

When debugging, read docs in this order:

1. `docs/COMMON-TROUBLESHOOTING.md` + the per-mode `TROUBLESHOOTING.md` (`follow/` or `snapshot/`) — fastest path to known traps
2. The per-mode `GUIDE.md` (`follow/GUIDE.md` or `snapshot/GUIDE.md`) — detailed setup and procedures
3. `README.md` — quick reference for common commands
4. `../../AGENTS.md` (repo root) — cross-network rules and secrets reference
