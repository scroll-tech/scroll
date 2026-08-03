# Follow Mode — Troubleshooting & Pitfalls

Follow-mode-specific traps (poll sync, starvation, live-mainnet finalization).
Mode-independent traps (environment, verifier deployment, relayer flags, prover setup) and the
step-by-step checklist live in [`../docs/COMMON-TROUBLESHOOTING.md`](../docs/COMMON-TROUBLESHOOTING.md);
see also the [Follow Mode Guide](./GUIDE.md). Trap numbers are stable across the split files.

## Critical Traps (Follow Mode)

### Trap 22: Poll-Synced Chunks Missing `l2_block` Linkage → "failed to fetch block hashes of a chunk" [follow mode]

- **Symptom**: In follow mode, after the initial backlog is proved, finalization stalls. Coordinator logs `format prover task failure ... failed to fetch block hashes of a chunk, chunk hash:0x... err:<nil>` on every dispatch attempt; provers spin on `CoordinatorEmptyProofData: get empty prover task` every ~20s; chunks pile up in `proving_status = 1` (with sweeps resetting stale `proving_status = 2` rows every 30 min).
- **Cause**: `sync-mainnet-db.py` poll mode copies new chunk/batch/bundle rows but historically skipped `l2_block` (watermarking the 181 GB mainnet table by `MAX(number)` times out). New chunks therefore had no block rows linked via `chunk_hash`, and the coordinator cannot format chunk tasks without them. Subtlety: the block rows usually **already exist** in the shadow DB (imported by block number during baseline) but with `chunk_hash = NULL`, so a plain `INSERT ... ON CONFLICT (number) DO NOTHING` copy is a no-op — the linkage must be established with an UPDATE.
- **Fix**: `sync_l2_blocks()` in `sync-mainnet-db.py` now runs every poll cycle: (1) `UPDATE l2_block SET chunk_hash = chunk.hash` for recent chunks (last 2000) whose blocks lack the link, then (2) copy any genuinely missing block rows from mainnet. One-off manual backfill if needed:
  ```sql
  UPDATE l2_block b SET chunk_hash = c.hash FROM chunk c
  WHERE b.number BETWEEN c.start_block_number AND c.end_block_number
    AND c.index > (SELECT COALESCE(MAX(index),0) - 2000 FROM chunk)
    AND (b.chunk_hash IS NULL OR b.chunk_hash <> c.hash);
  ```
- **Diagnosis query**: chunks lacking block linkage —
  ```sql
  SELECT count(*) FROM chunk c
  WHERE NOT EXISTS (SELECT 1 FROM l2_block b WHERE b.chunk_hash = c.hash);
  ```
- **Note**: `CoordinatorEmptyProofData` every 20s from an idle prover is the **normal** "no task available" poll response, not an error — chunk dispatch is serialized by parent-chunk proof dependency, so only 2-3 chunks prove concurrently even with 4 provers. It only indicates this trap when ALL provers spin AND the coordinator logs block-hash failures.
- **Secondary damage — attempt exhaustion starvation**: **Note (fixed upstream)**: this is now fixed in the coordinator itself — coordinator-side dispatch failures (task formatting, universal-task generation, prover-task insertion) refund the charged attempt via `orm.RefundAttemptsByHash` (`recoverAttempts` in `internal/logic/provertask/*_prover_task.go`), so `total_attempts` is no longer permanently burned by such failures. The sweeper's `total_attempts` reset described below is now belt-and-braces only. Historical description: the coordinator picks chunk tasks oldest-first but **skips tasks with `total_attempts >= 5`** (`chunk.go GetUnassignedChunk`: `total_attempts < maxAttempts`). Every failed dispatch during the outage burns one attempt, and `sweep-stale-proving.sh` historically reset only `proving_status`/`active_attempts`, not `total_attempts`. Result: the 44 backlog chunks hit 5/5 attempts and became permanently invisible — the coordinator silently leapfrogged to freshly-synced tip chunks while the backlog starved (symptom: only tip chunks prove, old pending chunks never get sessions, no ERROR logged). Fix: reset attempts after the root cause is repaired —
  ```sql
  UPDATE chunk SET total_attempts=0, active_attempts=0
  WHERE proving_status=1 AND total_attempts>0;
  ```
  (same shape for `batch`/`bundle`). `sweep-stale-proving.sh` now resets `total_attempts = 0` on every swept row so future outages self-heal.
- **Diagnosis query for starvation**: oldest pending chunk with maxed attempts —
  ```sql
  SELECT index, total_attempts, active_attempts FROM chunk
  WHERE proving_status = 1 ORDER BY index LIMIT 5;
  ```
- **Tertiary damage — NULL parent links block proof-status promotion**: poll sync copies chunk/batch rows with `ON CONFLICT DO NOTHING`, so rows copied *before* mainnet's proposer assigned them keep `batch_hash` / `bundle_hash` NULL forever. The coordinator_cron promotes `batch.chunk_proofs_status = Ready` only when all chunks joined via `chunk.batch_hash` are verified (`collect_proof.go checkBatchAllChunkReady`), and likewise `bundle.batch_proofs_status` via `batch.bundle_hash`. NULL links → promotion never happens → the batch/bundle is silently invisible to task selection (oldest-first query filters on the status flag; no ERROR is ever logged). Symptom: chunks all verified but batch sessions never start; one stray batch/bundle proves while its older siblings starve. Fix: re-derive links from the parent rows' index ranges —
  ```sql
  UPDATE chunk c SET batch_hash = b.hash FROM batch b
  WHERE c.index BETWEEN b.start_chunk_index AND b.end_chunk_index
    AND (c.batch_hash IS NULL OR c.batch_hash = '' OR c.batch_hash <> b.hash);
  UPDATE batch b SET bundle_hash = u.hash FROM bundle u
  WHERE b.index BETWEEN u.start_batch_index AND u.end_batch_index
    AND (b.bundle_hash IS NULL OR b.bundle_hash = '' OR b.bundle_hash <> u.hash);
  ```
  `sync-mainnet-db.py` runs both every poll cycle (`sync_parent_links()`, bounded to the recent 500 parents).

### Trap 23: Bundles Popping Post-Fork L1 Messages → VerificationFailed (0x439cc0cd), then ErrorFinalizedIndexTooLarge (0x16465978) [follow mode]

- **Symptom**: Most bundles finalize fine, but a bundle whose batches pop L1 messages enqueued **after the Anvil fork block** reverts on-chain with `VerificationFailed` (`0x439cc0cd`). After patching the queue hashes, it then reverts with `ErrorFinalizedIndexTooLarge` (`0x16465978`, appears as raw bytes `\x16FYx` in relayer logs).
- **Diagnosis flow (all verified during the 2026-07-13 incident, bundle 18070)**:
  1. Decode the bundle proof's `metadata.bundle_info` from the DB (`bundle.proof` is a JSON blob) and confirm every field matches mainnet DB values.
  2. Recompute `keccak256(abi.encodePacked(protocolVersion, publicInput))` with the wrapper's packed layout (`chainId 8B | msgQueueHash 32B | numBatches 4B | prevStateRoot | prevBatchHash | postStateRoot | batchHash | withdrawRoot`) and confirm it equals `metadata.bundle_pi_hash`.
  3. Call the wrapper's `verify(bundleProof, publicInput)` directly with the metadata-derived publicInput. **If it succeeds**, proof/digests are fine and the mismatch is contract-side — ScrollChain computes `messageQueueHash` itself via `L1MessageQueueV2.getMessageRollingHash()`.
  4. Compare `getMessageRollingHash(i)` on fork vs mainnet (public RPC e.g. `https://ethereum-rpc.publicnode.com` works; Alchemy demo key is 429-rate-limited).
- **Root cause**: `L1MessageQueueV2` keeps `mapping(uint256=>bytes32) messageRollingHashes` at **storage slot 101** (value = rolling hash with low 32 bits overwritten by enqueue timestamp; the getter clears those bits). Entries for messages enqueued after the fork block are zero on the fork, so the contract builds a publicInput whose `messageQueueHash` differs from the prover's → plonk rejects. Note the **off-by-one**: rh[i] is the hash *after* message i, so a bundle popping to index N needs rh[N-1]; DB `prev/post_l1_message_queue_hash` at popped_before=N equals `getMessageRollingHash(N-1)`.
- **Fix**: `scripts/sync-queue-hashes.py` copies `getMessageRollingHash(i)` from a mainnet RPC into the fork via `anvil_setStorageAt(queue, keccak256(pad32(i)‖pad32(101)), hash)` and also bumps `nextCrossDomainMessageIndex` (slot 103 = 0x67) to mainnet's value (this is what clears the follow-up `ErrorFinalizedIndexTooLarge`). It is idempotent and runs inside the `sync-mainnet-db.py` poll loop during follow mode (failures there only log a warning, never kill the DB sync). Run it manually after any Anvil restart/re-fork, and whenever a finalize reverts with either selector.
- **Rule of thumb**: if finalization fails only for bundles whose `post_l1_message_queue_hash != prev_l1_message_queue_hash` (i.e. the batch pops messages), suspect this trap before touching proofs or the verifier wrapper.

### Trap 25: Orphaned Poll-Sync + Empty Watermark → Full Mainnet History Backfill [follow mode]

- **Symptom**: Shadow DB suddenly fills with millions of mainnet rows (`sync-poll.log` shows `chunk: inserted 25000/6642569` and counting) even though the follow-mode baseline only covers the finalization-lag window (~3 chunks / 1 batch / 1 bundle).
- **Cause**: Two independent failures compound:
  1. **Orphaned daemon**: `11-follow-stop.sh` (or an earlier stop script) killed the loop `bash` process but not its `python3 sync-mainnet-db.py` child — the python process kept running in the background, invisible to pidfile checks.
  2. **Watermark = 0**: the DB was then reset (`--reset` TRUNCATEs the task tables). The orphaned sync's next poll computed `max(index) = 0` on the empty table and treated *all* of mainnet history as new rows, bulk-copying everything.
- **Defenses (all three are in place — do not remove any)**:
  1. `11-follow-stop.sh` kills the **whole process group** (`kill -- -$pid`, daemons are started with `setsid`), so python children die with their loop.
  2. `sync-mainnet-db.py` poll mode has a `MAX_POLL_DELTA = 5000` guard: if the watermark is 0 or the mainnet-vs-shadow delta exceeds 5000 rows, the poll **skips and warns** instead of backfilling. Bulk backfill is exclusively the baseline's job.
  3. `10-follow-up.sh` runs the baseline (`sync-baseline`) **before** starting any daemon, so a fresh DB always has a sane watermark before polling begins.
- **Recovery if it still happens** (e.g. you started a poll sync by hand against an empty DB): kill the sync process, then delete the contaminated rows below the baseline window:

  ```sql
  DELETE FROM l2_block WHERE number < <baseline_first_block>;
  DELETE FROM chunk    WHERE index  < <baseline_first_chunk>;
  DELETE FROM batch    WHERE index  < <baseline_first_batch>;
  ```

### Trap 26: Relayer L2 Watcher Genesis-Crawls an Empty `l2_block` [follow mode]

> **Fixed upstream**: the relayer now supports `l2_config.disable_l2_watcher` (default false = production behavior unchanged), which skips the watcher loop entirely. The shadow relayer config template (`lib/configs/relayer.json.template`) sets it — in follow mode the poll sync is what keeps `l2_block` populated for newly synced chunks, so the watcher is redundant. The harness ordering below remains as defense-in-depth.

- **Symptom**: `l2_block` fills with rows numbered 1, 2, 3, … at ~30–60 blocks/s, `chunk_hash = NULL`, `withdraw_root = 0x0000…` — a full L2 history crawl that would take days and tens of GB, while relayer logs show `retrieved block height=…` climbing from genesis.
- **Cause**: `rollup_relayer` always runs an L2 watcher loop (`rollup/cmd/rollup_relayer/app/app.go` → `TryFetchRunningMissingBlocks` every 2 s) that fetches **all** blocks from `MAX(l2_block.number)+1` up to the L2 tip — reading the max **once** at loop entry. If the relayer is alive while `--reset` TRUNCATEs `l2_block` (or the table is empty for any other reason), the watcher sees `MAX = 0` and starts crawling from block 1; the in-flight call keeps going for days even after the baseline repopulates the window.
- **Fix (in place)**: `10-follow-up.sh --reset` now runs `11-follow-stop.sh` **before** the TRUNCATE, so no writer survives into the reset. Never TRUNCATE `l2_block` (or let it go empty) while a relayer is running.
- **Recovery**: kill the relayer (stops the crawl), `DELETE FROM l2_block WHERE number < <window_first_block>`, restart the relayer — the watcher then computes `MAX` from the window rows and only fetches the small gap to tip, which is its normal, desirable behavior (keeps tip blocks present for newly synced chunks).

### Trap 27: Relayer Cannot Commit/Finalize on the Fork — Balance, Sequencer, Nonce Checklist [follow mode]

Three independent things must hold for the relayer to send its first on-fork commit; each fails with a distinct signature:

1. **Zero EOA balance** → `Out of gas: gas required exceeds allowance: 0`. `10-follow-up.sh`'s idempotent restart skips `01-setup-anvil.sh` when Anvil is already running, so EOA funding never re-runs. Fix: `cast rpc anvil_setBalance <eoa> 0x56bc75e2d63100000` for both the commit sender and the finalize sender.
2. **`ErrorCallerIsNotSequencer` (0x3cddbade)** on `commitBatches` → the commit sender is not authorized on the fork. `configs/mainnet.json` `accounts.commit_eoa` **must equal the address derived from the `COMMIT_KEY` hardcoded in `06-run-relayer.sh`** — they had drifted apart (`0xf39F…` vs `0xBC73…`), so `01-setup-anvil.sh` authorized an account the relayer never uses. Authorize manually: impersonate the ScrollChain owner and `cast send … "addSequencer(address)" <commit_eoa> --unlocked`.
3. **Tx stuck in txpool `queued` forever** → `pending_transaction` nonce desync (Trap 7 variant). The relayer's sender initializes its nonce as `max(db_max_nonce + 1, chain_pending_nonce)` (`rollup/internal/controller/sender/sender.go` `initializeNonce`). `pending_transaction` rows survive Anvil state-file restores, but the on-fork account nonce does not (a restored state can reset it to 0). The relayer then signs with a high nonce, Anvil queues the tx behind a gap that never fills, and *nothing ever mines* — commits silently stall with no error after the initial send. Diagnose with `cast nonce <eoa>` vs `cast rpc txpool_content`; fix: stop relayer, `DELETE FROM pending_transaction WHERE sender_address = '<eoa>'` (or all rows), restart relayer so it re-initializes from the chain nonce.

Also note: the relayer writes `rollup_status` **only** through its commit/finalize confirmation path (GORM `UPDATE … SET finalize_tx_hash, rollup_status`). If a batch/bundle shows `rollup_status = 5` with NULL `finalize_tx_hash` while `lastFinalizedBatchIndex` on the fork hasn't moved, do not trust the DB row — cross-check on-chain. `log_statement = 'mod'` on the shadow postgres is a cheap way to attribute every status write; it is asserted automatically by `02-prepare-db.sh` and `10-follow-up.sh` (re-applied on every setup, since `ALTER SYSTEM` lives in the container's data volume and is lost when the volume is recreated). Read the writes with `docker logs shadow-postgres` (or the postgres server log on a non-docker setup).


### Trap 28: Post-Fusaka Fork + Anvil 1.0.0 → Blob Base Fee Explosion, Commits Fail "Insufficient funds" [follow mode]

- **Symptom**: Right after a fresh fork, every `commitBatches` attempt fails with `estimateGasLimit failure ... Insufficient funds` (EOA balance is fine — verified 100 ETH). Batches sit at `rollup_status = 1` forever; `cast blob-base-fee` on the fork returns ~1e18 wei.
- **Cause**: Anvil 1.0.0 predates the Fusaka fork. Forking post-Fusaka mainnet state inherits a large `excessBlobGas` (~1.8e8 when the mainnet blob market is congested), but Anvil prices blob gas with the **Dencun** update fraction (3338477), so the inherited excess maps to an astronomical blob base fee. Commit txs carry blob versioned hashes and become unaffordable.
- **Fix (codified)**: `lib/01-setup-anvil.sh` now mines empty blocks right after `wait_for_anvil` until `cast blob-base-fee` drops below 1 gwei (excess decays ~1/8 per empty block; ~400 blocks suffice from 1.8e8). Manual equivalent: `cast rpc anvil_mine 400 --rpc-url http://localhost:18545`.
- **Note**: Forking at tip does NOT dodge this — it only depends on the fork block's `excessBlobGas`, which is high whenever mainnet blob backlog is high. Long-term fix is upgrading Anvil to a Fusaka-aware release.

### Trap 29: Same-Block Ordering — Manual `addProver` Mined After the Failing Finalize [follow mode]

- **Symptom**: A `finalizeBundlePostEuclidV2` tx lands on-chain but reverts with `ErrorCallerIsNotProver` (0x7b263b17) even though `addProver` was sent first; the bundle ends up at `rollup_status = 7` (RollupFinalizeFailed), which `ProcessPendingBundles` **never retries**.
- **Cause**: Anvil mines both txs in the same block and orders the finalize (txIndex 0) before the `addProver` (txIndex 1). The finalize legitimately reverts at execution time. (Also reachable via an Anvil state restore from a pre-addProver backup — see Trap 27.)
- **Fix (codified)**: `10-follow-up.sh` step h now re-ensures `isProver(finalize_sender)` idempotently on every run, alongside the Trap-27 balance/sequencer checks. Manual recovery: impersonate the owner, `cast send … "addProver(address)" <finalize_eoa> --unlocked`, verify `isProver` = true, then `UPDATE bundle SET rollup_status = 1 WHERE index = <n>` so the relayer retries.
- **Rule of thumb**: a bundle at `rollup_status = 7` is stranded by design (relayer logs it loudly). It always needs the manual `rollup_status = 1` reset after fixing the underlying cause.

### Trap 30: Fresh Shadow DB — Baseline Sync Fails, Schema Never Migrated [follow mode]

- **Symptom**: First-ever `10-follow-up.sh` on a brand-new shadow DB dies in step b: `sync-mainnet-db.py` aborts with `psql staging prep failed for chunk/batch/...: relation "public.<table>" does not exist`.
- **Cause**: `copy_table_pipe()` creates staging tables with `CREATE TABLE staging_x (LIKE public.x INCLUDING DEFAULTS)` — the real tables must already exist. Nothing in the follow bring-up migrates the schema before the baseline sync; the coordinator's auto-migration only happens later, when `coordinator_api` starts in step f. On the original author's machine the DB always carried schema from previous runs, so the gap went unnoticed.
- **Fix**: migrate once per fresh DB before first bring-up:
  ```bash
  cd database && go build -o /tmp/db_cli ./cmd
  /tmp/db_cli migrate --config <config-with-shadow-dsn>   # goose -> version 28
  ```

### Trap 31: Upgrade-Test Phase 1 Built From the Branch Under Test, Not From Production [follow mode]

- **Symptom**: Phase-1 chunks/batches prove fine, but every bundle finalize reverts with `VerificationFailed(0x439cc0cd)` even though `--skip-verifier` routing checks pass.
- **Cause**: Assuming the current checkout == the production zk stack. The branch under test and the config templates describe the NEW version (e.g. zkvm v0.9.0 / OpenVM 2.0.0), while mainnet may still run the previous guest (e.g. v0.8.0 / OpenVM 1.6.0 from `develop`). New-guest proofs can never verify against the production wrapper's digests — the mismatch only shows up at finalization, after hours of proving.
- **Fix**: determine the production stack FIRST (follow/GUIDE.md "Determining the Production zk Stack": on-chain wrapper `verifierDigest1/2()` vs S3 `digest_*.hex` — Montgomery conversion needed for v0.8.0, see docs/bundle-digest-encoding.md — plus `git_version` inside a production `bundle.proof` JSON, and `Cargo.lock` zkvm pins per branch). Build production in a separate `git worktree` and aim `COORD_DIR` / `PROVER_BIN` / `ASSETS_DIR` at it for Phase 1. Also remember the v0.8.0 S3 prefix has **no** `/releases/` segment.

### Trap 32: `cargo update -p scroll-zkvm-*` Drifts `revm`, Breaking the `[patch]` Fork Resolution [build]

- **Symptom**: After bumping the `scroll-zkvm-*` workspace pin (e.g. `tag = "v0.9.0"` → `rev = <master>`) and running `cargo update -p scroll-zkvm-prover ...`, the build fails deep in the dependency graph with `E0308 mismatched types ... expected revm_primitives::hardfork::SpecId, found SpecId` and the note "there are multiple different versions of crate `revm_primitives`" (crates.io vs the scroll `scroll-v91` fork).
- **Cause**: `cargo update -p` re-resolves more than the named crates. It flipped `alloy-evm 0.22.6`'s edge from `revm 30.1.1` to `revm 30.2.0` (and several `revm-primitives` edges from the patched fork `21.0.1` to crates.io `21.0.2`). The workspace `[patch.crates-io]` only redirects a revm crate to the scroll fork when the fork's version satisfies the requirement; the bumped crates.io `revm` family mixes fork and non-fork `revm-primitives` in one crate and cannot compile.
- **Fix**: restore the exact HEAD edges instead of letting the resolver choose. Inspect with `git diff Cargo.lock`; the two hand-edits that fixed it (2026-07, zkvm master bf887150): (1) in `alloy-evm 0.22.6`'s dependency list change `"revm 30.2.0"` back to `"revm 30.1.1"`; (2) in the six crates.io revm-family packages that flipped (`revm-context 10.1.2`, `revm-context-interface 11.1.2`, `revm-handler 11.2.0`, `revm-inspector 11.2.0`, `revm-interpreter 28.0.0`, `revm 30.2.0`) change `"revm-primitives 21.0.2"` back to `"revm-primitives 21.0.1"` (the fork). Verify with `cargo metadata --locked` before rebuilding. You cannot fix this with `cargo update -p revm --precise ...` — `op-revm` legitimately requires `revm ^30.2.0`, so both versions must coexist.
- **Prevention**: after any `cargo update -p scroll-zkvm-*`, always `git diff Cargo.lock` and revert every revm-family drift before building.

### Trap 33: v0.9.0+ Master Requires `agg_vk.bin` on S3 (Prover) and in Coordinator Assets [follow mode / upgrade]

- **Symptom**: New-stack prover exits at startup with `Failed to download agg_vk.bin: HTTP status 403` (or silently falls back to "deriving agg VK from SDK (slow, may allocate GPU memory)" and later OOMs the 24 GB card during SNARK proving). Coordinator panics with `agg_vk.bin missing from assets` when its first batch proof arrives.
- **Cause**: zkvm-prover master (bf887150, halo2-gpu) writes a per-circuit `agg_vk.bin` next to `app.vmexe` at `build-guest` time, and `Prover::load_agg_vk()` reads it to avoid constructing the GPU aggregation prover just to obtain the VK. The scroll prover downloads circuits from the **flat** S3 layout `<base>/<circuit>/app.vmexe` (no VK subdir) and expects `agg_vk.bin` in the same dir; the v0.9.0 S3 assets predate this file. The coordinator's batch-proof verification (deferral) reads the same key as `agg_vk.bin` from its own assets dir (`crates/libzkp/src/verifier/universal.rs`).
- **Fix**: after every guest rebuild that bumps the zkvm pin, upload the new artifacts (bucket layout is flat per circuit):
  ```bash
  Z=<zkvm-prover>/releases/dev
  B=s3://circuit-release/scroll-zkvm/releases/v0.9.0
  for c in chunk batch bundle; do aws s3 cp $Z/$c/agg_vk.bin $B/$c/agg_vk.bin; done
  # verify: anonymous GET must succeed (403 = wrong key or missing object)
  curl -s -o /dev/null -w '%{http_code}\n' https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/v0.9.0/chunk/agg_vk.bin
  ```
  For the local coordinator, copy the batch circuit's `agg_vk.bin` into `coordinator/build/bin/assets_v2/agg_vk.bin` (no separate S3 object needed — it is the same file the prover downloads from `batch/agg_vk.bin`).
- **Note**: `agg_vk.bin` contents are identical for batch and bundle (same agg config) and equal to `root_verifier_vk` for chunk — matching md5s are expected, not a copy/paste bug. And always re-run `make build-guest` after switching zkvm commits: a stale `releases/dev/` once produced a wrong `digest_1.hex` that only a fresh build corrected (digests must match the canonical values in docs/bundle-digest-encoding.md).

### Trap 34: Stale `prover_task` Failure Rows Starve Batch Assignment Silently [follow mode]

- **Symptom**: Chunks all prove, batches sit at `proving_status = 1` with `chunk_proofs_status = 2` (ready) forever, provers idle-poll `CoordinatorEmptyProofData: get empty prover task`, and the coordinator never logs `start batch proof generation session`. No ERROR anywhere.
- **Cause**: The coordinator's batch assignment returns the lowest-index unassigned batch (`ORDER BY index LIMIT 1`) and then applies the "don't dispatch the same failing job to the same prover" rule: if `prover_task` contains a `proving_status = 3` (ProverProofInvalid) row for that batch hash *and* the polling prover, the assignment silently returns empty — and because the query always picks the *same* lowest batch first, later batches are never considered. If every prover has a failure row for that one batch, all batch proving starves permanently. The sweeper (`sweep-stale-proving.sh`) resets `batch.proving_status`/`total_attempts` but does NOT clear `prover_task` failure rows, so the poison survives sweeps and coordinator restarts.
- **How it happens here**: a previous coordinator instance (e.g. a sanity run whose assets lacked `agg_vk.bin`, Trap 33) rejects a valid proof as `ProverProofInvalid`. The failure is the coordinator's fault, but the row pins the *prover* as having failed the task.
- **Fix**: delete the bogus failure rows (verify they are bogus first — `created_at` predating the current coordinator instance is a strong hint):
  ```sql
  SELECT task_type, task_id, prover_name, failure_type, created_at FROM prover_task WHERE proving_status = 3;
  DELETE FROM prover_task WHERE proving_status = 3 AND task_type = 2 AND task_id = '<batch_hash>';
  ```
- **Note**: task type is chosen at RANDOM per poll (`proofType()` in `get_task.go` shuffles chunk/batch/bundle), and a busy prover does not poll — so even a healthy system picks up batch tasks only on lucky idle polls; a few minutes of delay after unblocking is normal, hours is not.

### Trap 35: Post-Upgrade — Coordinator Loops "Generate universal prover task failure" on Old-Format Proofs [upgrade]

- **Symptom**: Right after the 20-upgrade.sh cutover, the new coordinator repeatedly logs `Generate universal prover task failure ... data did not match any variant of untagged enum ProofEnum` for the same one or two task ids, provers error on every poll (`CoordinatorGetTaskFailure`), and NO new tasks of any type get assigned (chunks included) — the whole fleet starves.
- **Cause**: three leftover-state problems stack up:
  1. **Stale assigned `prover_task` rows** — tasks that were in-flight at the cutover keep `proving_status = 1` (ProverAssigned) rows; the new coordinator's `hasAssignedTask` path rebuilds them from old-circuit child proofs, which the new libzkp cannot parse.
  2. **Stale ready flags on never-assigned rows** — a naive reset (`WHERE proving_status <> 1`) misses rows that were *already* unproved: bundles whose `batch_proofs_status = Ready` and batches whose `chunk_proofs_status = Ready` from old-circuit proving stay assignable, and their DB proof blobs are old-format.
  3. **Priority dispatch amplifies the poison** — the branch's GetTasks tries Bundle > Batch > Chunk and ABORTS the whole request when a higher-priority `Assign` errors (unlike develop's random pick), so one poisoned bundle blocks chunk/batch assignment too.
- **Fix (codified)**: 20-upgrade.sh step (e) now resets `batch_proofs_status`/`chunk_proofs_status` and clears `proof` blobs for ALL unfinalized rows at/after N (not just `proving_status <> 1`), and `DELETE FROM prover_task WHERE proving_status = 1` while the provers are stopped. Manual recovery is the same SQL; after cleanup the pipeline recovers within one poll cycle.
- **Related**: first chunk task generation after an assets swap is SLOW (the coordinator fetches `debug_executionWitness` block-by-block — ~550 blocks ≈ 10-15 min for two chunks — at 0% CPU). This is normal; the prover-side `connection_timeout_sec = 1800` covers it. Do not restart the coordinator just because it looks idle.

### Trap 36: `cast receipt <tx> status` Output Format Changed in Foundry ≥ 1.6 [tooling]

- **Symptom**: `03-deploy-verifier.sh` aborts with `updateVerifier reverted (tx ..., status 'true')` (or `status '1 (success)'`) even though the tx succeeded on-chain — `latestVerifier`/`getVerifier` already show the new wrapper.
- **Cause**: older cast prints `0x1`; foundry ≥ 1.6 prints `1 (success)`; foundry ≥ 1.7 prints `true`. A literal string comparison against `0x1`/`1` misreads success as failure. Same class of drift as the `cast blob-base-fee` removal (commit b4d41624).
- **Fix (codified)**: the status check now accepts `0x1`, `1`, `1 (success)`, and `true`. When in doubt, verify on-chain state instead of trusting the script's verdict: `cast call $MVRV "latestVerifier(uint256)(uint64,address)" 10 --rpc-url $RPC`.
- **Recovery when `20-upgrade.sh` dies at step h after a *successful* registration** (seen 2026-08-03, cast 1.7.1): do NOT re-run the whole script (it would try to register the same start batch again). Finish the remaining steps manually: (i) verify routing `getVerifier(10, N-1)` → old wrapper, `getVerifier(10, N)` → new; update `.work/verifier.env` to the new wrapper (the script died before writing it); (j) restart provers via `env GPUS=0,1 ASSETS_DIR=<new assets> lib/04-prover-up.sh --config mainnet-next`; (k) append `UPGRADE_AT_BATCH/UPGRADE_AT_TIME/UPGRADE_OLD_WRAPPER/UPGRADE_NEW_WRAPPER/UPGRADE_CIRCUIT_VERSION` to `.work/follow-run.env`.

### Trap 37: `make coordinator_api` Does NOT Refresh the Embedded `libzkp.so` [build / upgrade]

- **Symptom**: After rebuilding `target/release/libzkp.so` (e.g. for the `agg_vk.bin` verifier change) and restarting the coordinator, batch proof verification still runs the OLD code — valid proofs are rejected (`Batch verify failed, error: <old-asset-name> missing from assets`) and the rejections poison `prover_task` exactly like Trap 34.
- **Cause**: the coordinator Go binary CGO-links `coordinator/internal/logic/libzkp/lib/libzkp.so` — a **separate copy** that `make coordinator_api` does not rebuild or re-copy. Only `make -C coordinator libzkp` (or a manual `cp target/release/libzkp.so coordinator/internal/logic/libzkp/lib/`) refreshes it.
- **Fix**: after every libzkp-c rebuild that changes verifier/prover logic, sync the copy and restart the coordinators:
  ```bash
  cargo build --release -p libzkp-c
  cp target/release/libzkp.so coordinator/internal/logic/libzkp/lib/libzkp.so
  strings coordinator/internal/logic/libzkp/lib/libzkp.so | grep -c "<new-marker-string>"  # sanity check
  ```
  Then clear any `ProverProofInvalid` rows created while the stale .so was live (Trap 34 recovery SQL).

### Trap 38: halo2-gpu Bundle Prover Crashes — VRAM Starvation and `def_hook_commit` [upgrade / halo2-gpu]

- **Symptom 1**: Both provers die mid-bundle: `panicked ... called Result::unwrap() on an Err value: HaloGpu(Cuda(CudaError { code: 9, name: "cudaErrorInvalidConfiguration", ... quotient.cu }))` right after the halo2 `create_proof` phase, with the log showing `GPU mem ... peak=22.9 GiB`.
- **Cause 1**: scroll's prover-bin obtained the child aggregation VK via `sdk.agg_vk()`, which **builds the child's full GPU aggregation prover (~5.7 GiB per circuit)** just to read the VK. The openvm VPMM pool never returns those pages to the OS, so by SNARK time `cudaMemGetInfo` free ≈ 0 and the quotient chunking computes `batch_size = 0` → `cudaErrorInvalidConfiguration` (not a clean OOM). zkvm-prover master (bf887150) documents exactly this failure mode in its AGENTS.md "VRAM budgeting" section.
- **Fix 1**: `UniversalHandler::agg_vk()` now uses `Prover::load_agg_vk()`, which reads the pre-built `agg_vk.bin` asset (downloaded alongside `app.vmexe`) instead of materializing the GPU prover. Post-fix SNARK-phase peak dropped enough for 24 GB cards (observed halo2_outer ~8 s + wrapper ~2.3 s per bundle).
- **Symptom 2**: After fixing #1, the prover panics with `def_hook_commit must be defined to verify child proof with deferrals` as soon as a **bundle** task arrives before any batch task in a fresh process.
- **Cause 2**: the bundle verify circuit's `def_hook_commit` comes from the *batch child* SDK's deferral prover, which only exists after the batch prover's own `enable_deferral(chunk)` ran. Previously `sdk.agg_vk()` accidentally initialized it as a side effect; with the file-based `load_agg_vk()` that side effect is gone. (Batch tasks are unaffected: chunk children carry no deferral merkle proofs, so the assert passes with a `None` hook commit.)
- **Fix 2**: `do_prove` now initializes the batch child's deferral (`batch.enable_deferral(chunk_handler)`) before `bundle.enable_deferral(batch)` for bundle tasks — mirroring the zkvm integration tester's flow.
- **Symptom 3** (2026-08-03, post-Fix-1 binary): the *same* `quotient.cu ... invalid configuration` panic returns, peak ~22.8 GiB — but only when the prover process proved **chunk/batch tasks earlier in its lifetime** and then picks up a bundle task. A process that goes straight to bundle tasks (the post-Fix-1 verification scenario) survives.
- **Cause 3**: Fix 1 removed the 5.7 GiB/circuit agg-VK derivation, but the GPU circuit provers of every task type the process has served stay resident (VPMM pool never returns pages). Chunk+batch+bundle+halo2_outer together still exceed what the SNARK quotient chunking needs on a 24 GB card → `batch_size = 0` again. Mixed task types re-create the starvation Fix 1 only narrowed.
- **Workaround 3 (ops)**: pin task types per GPU so bundle proofs always run in a clean process — edit `.work/prover-<i>/prover.json` `sdk_config.prover.supported_proof_types` to `[1,2]` on one GPU and `[3]` on the other, then restart both provers.
- **Fix 3 (code, verified 2026-08-03)**: `crates/prover-bin` now calls `Prover::reset()` on the child/grandchild handlers right after `enable_deferral` (+ file-based `agg_vk()`) in `do_prove` — the child SDK exists only to configure the parent's deferral (agg_vk / cached commit / hook commit are captured by value), so its GPU proving keys are released before the parent's STARK/SNARK phase, mirroring the zkvm integration tester (`crates/integration/src/testers/bundle.rs`). Verified live on this shadow fork: bundle 18394 was proven by a **mixed-type process** (same process had proved chunk tasks minutes earlier, `supported_proof_types=[1,2,3]`) and finalized via the new wrapper — halo2_outer `create_proof` live peak was **8.6 GiB** (vs 22.8 GiB in the crash), and the quotient phase ran with `current` 6.4–8.6 GiB instead of 14.7–22.7 GiB. Note the VPMM `in pool=` number still shows the historical high-water mark (~22.4 GiB from earlier chunk proving) — the pool never shrinks, but its free regions are reusable; what matters for the quotient chunking heuristic is that *physical* free stays > ~256 MiB, which on 24 GB cards remains a thin margin.
- **Root-cause note for upstream**: halo2-axiom-gpu's `query_device_free_bytes_for_chunking()` budgets from raw `cudaMemGetInfo` (physical free) while its own allocations go through the VPMM pool — pool-internal free regions are usable but not counted. A minimal upstream fix: add `MemoryManager::pool_free_bytes()` (sum of `free_regions`) to openvm-cuda-common and budget `physical_free + pool_free_bytes() - RESERVED` in `cuda/utils.rs`.
- **Operational note**: after a prover crash, also `DELETE FROM prover_task WHERE proving_status = 1` and reset the affected `bundle`/`batch` rows — a task proved from a *deleted* assignment row is rejected with `validator failure get none prover task for the proof`, and the SDK will happily re-prove its locally-cached stale task instead of picking up the fresh assignment. Three corollaries learned on 2026-08-03:
  1. **Also reset `active_attempts = 0`** on the reset `bundle`/`batch` rows. `GetUnassignedBundle` requires `active_attempts < max`, and once the `prover_task` rows are deleted, coordinator-cron's `DecreaseActiveAttemptsByHash` matches nothing ("No rows were affected") — the row sits at `active_attempts = 1` forever and is never re-dispatched, with no ERROR logged.
  2. **Wipe the crashed prover's local SDK db** (`.work/prover-<i>/db`) *before* restarting it, or it loops forever on the cached task: build fails (`unsupported task type` if you also narrowed `supported_proof_types`) and every submit is rejected (`get none prover task`).
  3. Restart order: SQL cleanup first, then db wipe, then prover restart — restarting against a dirty SDK db just re-poisons the loop.
