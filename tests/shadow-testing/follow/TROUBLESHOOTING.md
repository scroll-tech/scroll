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

