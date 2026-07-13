# Snapshot Replay Mode — Troubleshooting & Pitfalls

Snapshot-replay-mode-specific traps (historical fork, fixed bundle range, Sepolia).
Mode-independent traps (environment, verifier deployment, relayer flags, prover setup) and the
step-by-step checklist live in [`../docs/COMMON-TROUBLESHOOTING.md`](../docs/COMMON-TROUBLESHOOTING.md);
see also the [Snapshot Replay Mode Guide](./GUIDE.md). Trap numbers are stable across the split files.

## Critical Traps (Snapshot Replay Mode)

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
  `end_block_number`.

### Trap 18: Relayer Creates Competing Batches [snapshot mode]

- **Symptom**: Provers are busy but your target bundles make little progress; coordinator assigns batch/bundle tasks with indices far beyond your target range.
- **Cause**: The relayer keeps proposing new batches from live L2 blocks. Those batches become valid proving tasks and consume prover time.
- **Rule**: If you only need to prove a fixed imported range, **stop the relayer** after the range is imported and committed. Restart it once proofs are ready for finalization.

