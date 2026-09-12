# Snapshot Replay Mode — Troubleshooting & Pitfalls

Snapshot-replay-mode-specific traps (historical fork, fixed bundle range, Sepolia).
Mode-independent traps (environment, verifier deployment, relayer flags, prover setup) and the
step-by-step checklist live in [`../docs/COMMON-TROUBLESHOOTING.md`](../docs/COMMON-TROUBLESHOOTING.md);
see also the [Snapshot Replay Mode Guide](./GUIDE.md).

- **Registry & statuses**: [`../docs/TROUBLESHOOTING.md`](../docs/TROUBLESHOOTING.md) — every trap's status (Active/Fixed/Checklist) and key symptom.
- **Retired traps** (e.g. 51, multi-prover cache conflict — fixed by per-GPU work dirs): [`../docs/TRAP-ARCHIVE.md`](../docs/TRAP-ARCHIVE.md). Numbers are retired, never reused.

## Critical Traps (Snapshot Replay Mode)

### Fork State & Contract Access

### Trap 4: `committedBatches` Sparse (EuclidV2) [snapshot mode]
- **Symptom**: `ErrorIncorrectBatchHash(0x2a1c1442)`.
- **Cause**: EuclidV2 only stores the **last batch hash** of each commit tx. Intermediate batches have `committedBatches[index] = 0x0`.
- **Rule**: The contract checks `committedBatches[batchIndex]` where `batchIndex` is the **end batch** of the bundle. Verify this is non-zero before finalizing.

### Trap 5: `L1MessageQueueV2` Index Mismatch (Sepolia) [snapshot mode]
- **Symptom**: `ErrorFinalizedIndexTooLarge(0x16465978)` or `ErrorFinalizedIndexTooSmall`.
- **Cause**: `nextUnfinalizedQueueIndex` does not match the pre-finalization state expected by the first target batch. The fork block is AFTER real finalization, so the real state has post-finalization values.
- **Rule**:
  1. Set `nextUnfinalizedQueueIndex` to `MIN(total_l1_messages_popped_before)` of the first target batch's chunks (from DB).
  2. **Use `forge inspect L1MessageQueueV2 storage-layout`** to find the exact storage slot (it's slot 104, NOT slot 0 or 4, due to OpenZeppelin `__gap`).
  3. Never guess storage slots from source code.

### Trap 54: `anvil_setStorageAt` Is Ignored on Mapping Slots [both]
- **Symptom**: storage patched via `anvil_setStorageAt` is visible to `eth_getStorageAt` but contracts still read the old value during `eth_call` / `eth_sendTransaction`.
- **Cause**: Anvil bug — mapping-slot writes (e.g. `committedBatches[i]`, `finalizedStateRoots[i]`) are cached and ignored by the EVM; **direct variable slots** (`miscData` slot 0xa1, `nextUnfinalizedQueueIndex` slot 0x68, etc.) do work.
- **Rule**: override simple variables freely; for mappings, fork at a block where the desired state already exists (or use a mock).

### Trap 55: Anvil's Default Account Is Not an EOA in Fork Mode [both]
- **Symptom**: `addProver(0xf39F…)` reverts with `ErrorAccountIsNotEOA`.
- **Cause**: Anvil's default test account carries contract code (`0xef0100…`, EIP-7702 delegation) in fork mode; `ScrollChain.addProver` requires `code.length == 0`.
- **Rule**: use a freshly generated EOA (`cast wallet new`) as prover/finality sender. If the fork's real owner is a 7702-delegated account, temporarily swap the proxy owner (slot 51), call `addProver`, then restore.

### Import & DB Hygiene

### Trap 6: Parent Batch Missing [snapshot mode]
- **Symptom**: Relayer logs `Batch.GetBatchByIndex error: record not found, index: <parent>`.
- **Cause**: Shadow DB imported bundles starting at batch N, but batch N-1 was not imported.
- **Rule**: Always insert the parent batch skeleton before starting the relayer. Only `state_root` must be accurate.

### Trap 8: Production Proof Overwrite [snapshot mode]
- **Symptom**: `VerificationFailed` after importing production data.
- **Cause**: Import script copied `proof` columns from production RDS, overwriting locally-valid shadow proofs.
- **Rule**: **Never import `proof` columns**. Import only metadata, then reset `proving_status = 1` and re-prove locally.

### Trap 15: Slow `l2_block` Export by `chunk_hash` JOIN [snapshot mode]
- **Symptom**: `00-import-bundle-range.sh` hangs for minutes on the `l2_block` export (0-byte CSV) — the
  `l2_block ⋈ chunk ON chunk_hash` JOIN full-scans the huge prod table.
- **Rule**: export `l2_block` by **block-number range** instead:
  `COPY (SELECT * FROM l2_block WHERE number BETWEEN <min_start_block> AND <max_end_block>) TO STDOUT …`
  (PK-indexed, seconds). Derive the range from the target batches' chunks' `start_block_number` /
  `end_block_number`. Always include `deleted_at IS NULL` in src-side queries ([`../docs/rds-query-rules.md`](../docs/rds-query-rules.md)).

### Trap 52: Orphan Bundles Deadlock Bundle Assignment [snapshot mode]
- **Symptom**: chunks/batches prove fine but **no bundle task is ever assigned**; coordinator loops silently.
- **Cause**: the `bundle` table retains historical rows (e.g. index 20000+) whose batches were never imported (`batch` only holds the recent range). `GetUnassignedBundle` picks the lowest-index bundle with `batch_proofs_status = 2`, finds no linked batches, and fails silently in a loop.
- **Diagnosis / fix**:
  ```sql
  SELECT COUNT(*) FROM bundle b WHERE NOT EXISTS (
    SELECT 1 FROM batch bat WHERE bat.index BETWEEN b.start_batch_index AND b.end_batch_index);
  UPDATE bundle SET batch_proofs_status = 1 WHERE index NOT IN (
    SELECT DISTINCT b.index FROM bundle b
    JOIN batch bat ON bat.index BETWEEN b.start_batch_index AND b.end_batch_index);
  ```

### Trap 53: Imported Rows with `proving_status = 2` and `proof IS NULL` Poison Batch Status [snapshot mode]
- **Symptom**: coordinator marks `batch.chunk_proofs_status = 2` but batch task formatting then fails.
- **Cause**: imported chunks kept an "assigned" status without proofs.
- **Fix**:
  ```sql
  UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0
   WHERE proving_status = 2 AND proof IS NULL;
  UPDATE batch SET chunk_proofs_status = 0
   WHERE chunk_proofs_status != 0 AND EXISTS (
    SELECT 1 FROM chunk c WHERE c.batch_hash = batch.hash AND c.proving_status != 4);
  ```

### Pipeline & Relayer

### Trap 18: Relayer Creates Competing Batches [snapshot mode]
- **Symptom**: Provers are busy but your target bundles make little progress; coordinator assigns batch/bundle tasks with indices far beyond your target range.
- **Cause**: The relayer keeps proposing new batches from live L2 blocks. Those batches become valid proving tasks and consume prover time.
- **Rule**: If you only need to prove a fixed imported range, **stop the relayer** after the range is imported and committed. Restart it once proofs are ready for finalization.

### Trap 49: Coordinator Healthy but Prover Gets No Tasks — Chunk Data Prerequisites [snapshot mode]
- **Symptom**: `Start coordinator api successfully` logged, prover polls but every response is empty.
- **Cause** (check in order): (1) `l2_block` has no rows for the chunk's block range (or rows lack the `chunk_hash` link); (2) chunk `proving_status <> 1`; (3) chunk `codec_version = 5` — such chunks are skipped by design; (4) chunk `end_block_number` above the coordinator's known L2 height.
- **Rule**: populate + link `l2_block` (`fetch-l2-blocks.py` then the `chunk_hash` UPDATE), reset proving status, and pick chunks within the imported block range.

### Trap 50: "mismatched post-state root" — Fork Block / Codec Version Mismatch [both]
- **Symptom**: Proving (usually chunk/task generation) fails with `mismatched post-state root`, or E2E import fails with codec errors.
- **Cause**: Blocks predate the hardfork matching the configured codec. GalileoV2 = codec V10 → mainnet blocks must be **≥ 33,750,000**; older blocks (e.g. 26,653,680) are Galileo / codec V9.
- **Rule**: check `codec_version` in the scenario/config and that `SCROLL_FORK_NAME` / the coordinator's verifier fork list matches the block era. Fork/block thresholds per codec live in [`../docs/CURRENT-STACK.md`](../docs/CURRENT-STACK.md).

### Trap 56: Relayer Finalize Sends a Stale `batch.withdraw_root` [snapshot mode]
- **Symptom**: `VerificationFailed(0x439cc0cd)` at finalize although verifier digests and MVRV routing are all correct.
- **Cause**: `fetch-l2-blocks.py` cannot read withdraw roots from a standard L2 RPC, so `l2_block.withdraw_root` (and downstream `chunk`/`batch.withdraw_root`) hold `0x0…0` placeholders. The contract folds the passed `withdrawRoot` into the public-input hash, which then mismatches the proof's `bundle_pi_hash`.
- **Fix (relayer)**: pack the finalize calldata with `proof.metadata.bundle_info.withdraw_root`, not `dbBatch.WithdrawRoot`.
- **Fix (shadow DB)**: after batch proofs verify, sync the column from proof metadata:
  ```bash
  DB_DSN="postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup" \
    python3 snapshot/scripts/09-sync-batch-withdraw-roots.py --batch-range <start:end>
  ```
- **Production note**: mainnet populates `l2_block.withdraw_root` correctly; the relayer change is defensive, the sync script is shadow-only.
