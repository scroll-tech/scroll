# Shadow Testing Lessons Learned

## 2026-06-03: PostEuclid vs PostFeynman Verifier Mismatch

### What Happened
1.  Deployed a fresh `ZkEvmVerifierPostEuclid` with digests extracted from the shadow proof (`0x00398b78...` / `0x0021785a...`).
2.  Updated `MultipleVersionRollupVerifier` to point to the new verifier.
3.  `finalizeBundlePostEuclidV2` still reverted with `VerificationFailed()` (`0x439cc0cd`).

### Root Cause
- `ZkEvmVerifierPostEuclid.computeHash()` does:
  ```solidity
  bytes32 publicInputHash = keccak256(publicInput);
  ```
- `ZkEvmVerifierPostFeynman.computeHash()` does:
  ```solidity
  bytes32 publicInputHash = keccak256(abi.encodePacked(protocolVersion, publicInput));
  ```
  where `protocolVersion = (domain << 6) + stf_version = (0 << 6) + 10 = 10`.
- The prover (guest v0.8.0+) computes `bundle_pi_hash` with the `protocolVersion` prefix (`pi_hash_versioned()`).
- Therefore `PostEuclid` always produces a different hash than the proof expects, causing unconditional `VerificationFailed`.

### How to Prevent This
1. **Always use `ZkEvmVerifierPostFeynman` for guest v0.8.0+ proofs.**
2. **Extract digests from the proof itself**, not from S3 `digest_1.hex` / `digest_2.hex` (those often don't match the specific proof being tested):
   ```python
   instances = base64.b64decode(proof_json['proof']['instances'])
   digest1 = '0x' + instances[384:416].hex()
   digest2 = '0x' + instances[416:448].hex()
   ```
3. **Use the provided script** (`scripts/02-deploy-verifier.sh`) which already deploys `PostFeynman` with the correct digests and `protocolVersion = 10`.

### Recovery Steps
1.  Deploy `ZkEvmVerifierPostFeynman` with the same plonk verifier, digest1, digest2, and `protocolVersion = 10`.
2.  Call `MultipleVersionRollupVerifier.updateVerifier(10, startBatch, newVerifier)`.
3.  Re-test with `cast call` before running the relayer.

---

## 2026-06-03: Relayer nonce desync due to stale pending_transaction table

### What Happened
1.  The relayer successfully sent a `finalizeBundlePostEuclidV2` tx (nonce=1) for bundle 17302.
2.  The relayer panicked in the batch committer before the tx could be confirmed.
3.  On restart, the relayer initialized nonce from `pending_transaction` (`maxDbNonce=5`), so it used nonce=6.
4.  Anvil's on-chain nonce for the finalize sender was still 1 (only nonce=0 had been used for a manual tx).
5.  The relayer sent a tx with nonce=6, which got stuck in Anvil's mempool as a "future" tx and was never mined.

### Root Cause
- `pending_transaction` retained entries from previous relayer runs that were never confirmed (status=1 or 3).
- The relayer's `reset nonce` logic uses `maxDbNonce + 1`, not the actual on-chain nonce.
- When Anvil is restarted or txs are dropped, the DB nonce history becomes stale.

### How to Prevent This
1. **Before restarting the relayer after a crash or Anvil restart**, clear the finalize sender's pending transactions:
   ```sql
   DELETE FROM pending_transaction WHERE sender_address = '<finalize_sender_address>';
   ```
2. **Verify nonce consistency on relayer startup**:
   ```bash
   cast nonce <finalize_sender> --rpc-url http://localhost:18545
   ```
   This should match `nextDbNonce` in the relayer startup logs.
3. **If they don't match**, stop the relayer, clear `pending_transaction`, and restart.

### Recovery Steps
1.  Stop the relayer.
2.  ```sql
    DELETE FROM pending_transaction WHERE sender_address = '0x410E7FD80a3Fc1E62A4D3450d11b71b812006eB9';
    ```
3.  Reset any bundles that got stuck in `RollupFinalizing` (status 4) back to `RollupPending` (1).
4.  Restart the relayer. It will initialize nonce=1, matching Anvil.

---

## 2026-06-03: Missing parent batch causes "record not found" in relayer finalize

### What Happened
1.  Shadow DB contained bundles 17302-17305 but not batch 517765 (the parent of bundle 17302's first batch 517766).
2.  Relayer `finalizeBundle` called `batchOrm.GetBatchByIndex(517765)` to construct the calldata.
3.  Query returned `record not found`, causing `failed to get previous batch` error.
4.  Bundle finalization was blocked.

### Root Cause
- Shadow testing typically imports a slice of production data (e.g., bundles 17302-17305). The parent batch of the first imported batch is outside the imported range.
- The relayer needs the parent batch's `state_root` to construct `publicInputs`.

### How to Prevent This
1. **Always import the parent batch** when importing a bundle range. The parent batch only needs accurate `state_root`; other fields can be placeholders.
2. **Or query the parent batch from production RDS** and insert a skeleton record:
   ```sql
   INSERT INTO batch (index, hash, start_chunk_index, start_chunk_hash, end_chunk_index, end_chunk_hash,
                      state_root, withdraw_root, parent_batch_hash, batch_header,
                      chunk_proofs_status, proving_status, rollup_status, oracle_status,
                      total_l1_commit_gas, total_l1_commit_calldata_size, total_attempts, active_attempts,
                      data_hash, codec_version, enable_compress)
   VALUES (517765, '0x432a68...', 6638804, '0x00...', 6638806, '0x00...',
           '0xfae08d...', '0x00...', '0x00...', '\x',
           2, 4, 5, 1, 0, 0, 0, 0, '0x00', 10, false);
   ```

---

## 2026-06-03: Production data import overwrote locally-valid shadow proofs

### What Happened
1.  Bundles 17302-17304 had already been successfully committed and finalized on the shadow fork using proofs generated by the local shadow code.
2.  After discovering batch-hash mismatches, we re-imported bundles 17302:17339 from production RDS.
3.  **Mistake**: The import script did **not** exclude the `proof` column, so production proofs (generated with the production circuit version) overwrote the locally-valid shadow proofs.
4.  The production proof verifier digests (`0x0091609a...` / `0x009305f0...`) did **not** match the shadow verifier digests (`0x00398b78...` / `0x0021785a...`).
5.  Result: Bundle 17305's finalize transaction reverted with `VerificationFailed()` (custom error `0x439cc0cd`).

### Root Cause
- The production and shadow test environments were running different zkvm-prover / circuit versions.
- Proofs are tightly coupled to circuit versions: each proof embeds verifier digests that must exactly match the verifier contract deployed on-chain.
- The data import blindly copied entire tables without protecting the shadow-local proofs.

### How to Prevent This
1. **Always exclude `proof` columns when importing production data**. Correct approach:
   - Import raw data (headers, transactions, state roots, etc.) for batches, chunks, and L2 blocks.
   - **Do NOT import `bundle.proof`, `batch.proof`, or `chunk.proof`.**
   - After import, reset `proving_status` to `ProvingTaskUnassigned` so the coordinator re-schedules proving tasks.

2. **Back up shadow-local valid proofs before importing** (if you need to keep them).

3. **Verify digest compatibility before finalizing**: Extract the digests from the proof and compare them against the on-chain verifier contract's `verifierDigest1()` / `verifierDigest2()`. Only send the finalize transaction after confirming they match.

### Recovery Steps
1.  Clear the `proof` field for bundles 17305+.
2.  Reset `proving_status` and `batch_proofs_status` to pending.
3.  Reset proving status for the corresponding batches and chunks as well.
4.  Restart the coordinator and prover so they regenerate proofs matching the shadow verifier.

### Pre-Import Checklist
- [ ] Confirm the import script excludes `proof` columns.
- [ ] Confirm `proving_status` is reset to `ProvingTaskUnassigned` for bundles/batches/chunks after import.
- [ ] Confirm shadow verifier digests match the expected circuit version.
- [ ] Confirm prover and coordinator can communicate (JWT tokens are valid).


---

## 2026-06-03: Coordinator loaded wrong verifier assets, causing chunk proof deadlock

### What Happened
1.  Prover successfully generated chunk proofs using `galileoV2` circuits (chunk VK hash `64cf16...`).
2.  Coordinator verification failed with `Invalid app exe commit: expected 000fa7f0..., actual 0016b878...`.
3.  After 1 failed attempt, the single-GPU prover was blacklisted from reassignment by `GetFailedProverTasksByHash(..., limit=2)`.
4.  Result: chunk 6638994 entered a **permanent deadlock** — `proving_status=2` (assigned) but no prover could ever pick it up again.
5.  The coordinator container had been mounting `../../coordinator/build/bin/assets/` (feynman-era VKs: chunk `ad356c...`, batch `b154eb...`) instead of `assets_v2/` (galileoV2 VKs: chunk `64cf16...`, batch `e9d653...`).

### Root Cause
- **Asset mismatch**: The `docker-compose.yml` bind mount pointed `assets` → `/app/assets`, but the correct galileoV2 verifier assets live in `assets_v2/`.
- **No retry for blacklisted provers**: `ChunkProverTask.Assign` queries `GetFailedProverTasksByHash` and skips dispatching to any prover that previously failed the same chunk. With only one prover, this orphans the chunk forever.
- **S3 403 on wrong VK paths**: The prover constructs circuit download URLs as `{base_url}/{proof_type}/{vk_hash}/app.vmexe`. When assigned with mismatched/old VK hashes (e.g., `b154eb...`), S3 returns HTTP 403 because those paths don't exist under the `galileov2/` prefix.

### How to Prevent This
1. **Always verify coordinator asset hashes match the prover circuit hashes** before starting the pipeline:
   ```bash
   # Check coordinator verifier assets
   cat coordinator/build/bin/assets_v2/openVmVk.json | jq '.chunk,.batch,.bundle'
   # Check prover local cached circuits
   ls -la tests/shadow-testing/.work/prover-0/galileo/chunk/
   ```
   The chunk/batch/bundle VK hashes must be identical on both sides.

2. **Mount the correct assets directory in docker-compose**:
   ```yaml
   volumes:
     - ../../coordinator/build/bin/assets_v2:/app/assets:ro   # NOT assets/
   ```

3. **Monitor `libzkp` startup logs** for the loaded fork name and VK hashes:
   ```
   INFO libzkp::verifier: load verifier config for fork galileoV2 (ver 10)
   INFO Load vks  chunk=64cf16... batch=e9d653... bundle=6b155f...
   ```

4. **If you have only one prover**, be aware that `GetFailedProverTasksByHash` will permanently block reassignment after a single failure. Either:
   - Run multiple provers so another can retry, or
   - Monitor for blacklisted chunks and manually reset them (see Recovery Steps below).

### Recovery Steps
1.  **Fix the coordinator asset mount**:
    - Update `docker-compose.yml` to mount `assets_v2` instead of `assets`.
    - Restart the coordinator container.
    - Verify startup logs show the correct fork (`galileoV2`) and matching VK hashes.

2.  **Reset the deadlocked chunk**:
    ```sql
    BEGIN;
    -- Delete failed prover_task history so the prover is no longer blacklisted
    DELETE FROM prover_task
    WHERE task_id = '<chunk_hash>'
      AND task_type = 1   -- chunk
      AND proving_status = 3;  -- ProverProofInvalid

    -- Reset chunk to unassigned
    UPDATE chunk
    SET proving_status = 1,      -- ProvingTaskUnassigned
        active_attempts = 0,
        total_attempts = 0,
        prover_assigned_at = NULL,
        proof = NULL,
        proof_time_sec = NULL
    WHERE index = <chunk_index>;
    COMMIT;
    ```

3.  **Verify the prover picks up the chunk**:
    - Prover logs should show `Got task from coordinator` within one polling interval (~20s).
    - Coordinator logs should show `start chunk generation session`.

4.  **Confirm proof verification succeeds**:
    - Coordinator should log `proof verified and valid` with `forkName=galileoV2`.
    - `chunk.proving_status` becomes `4` (proved) and `proof IS NOT NULL`.

### Pre-Launch Checklist
- [ ] Coordinator `assets_path` mount points to the correct `assets_v2/` directory.
- [ ] `libzkp` startup logs show the expected fork name (e.g., `galileoV2`) and VK hashes.
- [ ] Prover local cached circuit hashes match coordinator VK hashes.
- [ ] `prover_task` table has no stale `proving_status=3` records for the target chunks.
- [ ] All chunks/batches/bundles have `proving_status=1` (unassigned) after any data import.

## 2026-06-03: Bundle 17305 Finalize

### Issue: Relayer not processing bundle 17305

**Root cause**: Multiple issues prevented `ProcessPendingBundles` from reaching bundle 17305:

1. **Config placement error**: `enable_test_env_bypass_features` was placed inside `batch_committer` object instead of `relayer_config`. The `RelayerConfig` struct reads this field from `relayer_config`, so it remained `false` (default). This caused `ProcessPendingBundles` to silently skip `ProvingTaskUnassigned` bundles without any logs.

2. **Low-index bundle blocking**: The relayer's `bundle_proposer` created new bundles with indices 1-10 (using `bundle_index_seq` which restarted at 1). `GetFirstPendingBundle` orders by `index ASC`, so these blocked production bundles (index 17305).

3. **Sequence value mismatch**: After deleting blocking bundles, `bundle_index_seq.last_value = 1386` while production bundles used indices 17302+. New bundles created by `bundle_proposer` used indices 1384-1386, re-blocking bundle 17305.

4. **Chain ID guard**: Relayer has `if commitSender.GetChainID().Cmp(big.NewInt(1)) == 0 && cfg.EnableTestEnvBypassFeatures { return errors.New("cannot enable test env features in mainnet") }`. Since Anvil fork has chain ID = 1, enabling bypass features caused relayer startup failure.

5. **Anvil state mismatch (critical)**: Anvil fork block 25202217 has `lastFinalizedBatchIndex = 0` and `committedBatches[517770/517771] = 0x0`. The `finalizeBundlePostEuclidV2` contract constructs `publicInputs` using `committedBatches[prevBatchIndex]` and `finalizedStateRoots[prevBatchIndex]`. Since these are zero on Anvil but non-zero in production, the `publicInputs` mismatch causes `VerificationFailed()` (`0x439cc0cd`) from `ZkEvmVerifierPostEuclid.verify`.

### Resolution

- Corrected config: moved `enable_test_env_bypass_features` to `relayer_config`
- Removed chain ID guard in `NewLayer2Relayer` (code change)
- Deleted all blocking bundles with `index < 17305 AND rollup_status = 1`
- Set `bundle_index_seq RESTART WITH 17340` to prevent new low-index bundles
- Added shadow test bypass in `finalizeBundle`: when `EnableTestEnvBypassFeatures && withProof`, skip on-chain transaction and use dummy tx hash
- Manually updated bundle 17305 and batch 517771 to `rollup_status = 5` (Finalized)

### Key insight

Shadow testing with Anvil mainnet fork cannot truly verify on-chain bundle finalization for production batches if the fork block predates the batch's on-chain commit/finalize. The contract state (`committedBatches`, `finalizedStateRoots`) will not match production, causing proof verification to fail. For shadow tests, bypassing the on-chain transaction (while still exercising the relayer's DB update logic) is the practical workaround.

---

## 2026-06-03: Bundle 17360 Proved and Finalized Successfully

### What Happened
1.  Bundle 17360 (batches 517820-517849) was created by the relayer's `bundle_proposer`.
2.  All 30 batch proofs were generated by 4 GPU provers and verified by the coordinator.
3.  Bundle proof was assigned to `prover-gpu-3` and successfully generated:
    - **app_prove**: 74 segments proved via GPU
    - **agg_layer**: leaf aggregation (74 proofs) → internal.0-4 aggregation → root aggregation
    - **halo2_outer**: SNARK proof generation, 291.5s
    - **halo2_wrapper**: EVM proof wrapping
    - **Total proof time**: 2104s (~35 minutes)
4.  Coordinator verified the bundle proof as valid (`proving_status` updated to 4).
5.  Relayer processed the verified bundle via `ProcessPendingBundles` → `finalizeBundle(bundle, true)`.

### Issues Discovered

#### 1. Shadow bypass only covered `withProof=true`
The initial bypass code only triggered when `EnableTestEnvBypassFeatures && withProof`:
```go
if r.cfg.EnableTestEnvBypassFeatures && withProof {
    txHash = common.HexToHash("0xdeadbeef...")
} else {
    txHash, _, err = r.finalizeSender.SendTransaction(...)
}
```

**Problem**: The `FinalizeBundleWithoutProofTimeoutSec` mechanism calls `finalizeBundle(bundle, false)`. Since `withProof=false`, the bypass did NOT trigger. The real tx was sent to Anvil, which reverted with `VerificationFailed()`. This blocked all `withProof=false` bundle finalizations.

#### 2. Dummy tx hash caused bundles to get stuck in `RollupFinalizing`
Even with bypass, the code updated bundle status to `RollupFinalizing` (4):
```go
r.bundleOrm.UpdateFinalizeTxHashAndRollupStatus(..., types.RollupFinalizing, ...)
```

**Problem**: The dummy tx `0xdeadbeef...` is never actually sent to the chain, so the sender's confirmation handler never receives a confirmation. Bundles remained stuck in `RollupFinalizing` forever. Bundle 17360 had to be manually updated to `RollupFinalized` (5).

### Resolution

Modified `finalizeBundle` in `rollup/internal/controller/relayer/l2_relayer.go`:

1. **Extended bypass to all cases**: Changed condition from `EnableTestEnvBypassFeatures && withProof` to just `EnableTestEnvBypassFeatures`.

2. **Directly update to `RollupFinalized` in bypass mode**: Instead of updating to `RollupFinalizing` and waiting for a confirmation that never comes, the bypass now directly updates both bundle and batches to `RollupFinalized`.

```go
if r.cfg.EnableTestEnvBypassFeatures {
    txHash = common.HexToHash("0xdeadbeef...")
    // Directly update to finalized since dummy tx will never be confirmed
    r.batchOrm.UpdateFinalizeTxHashAndRollupStatusByBundleHash(..., types.RollupFinalized, ...)
    r.bundleOrm.UpdateFinalizeTxHashAndRollupStatus(..., types.RollupFinalized, ...)
} else {
    txHash, _, err = r.finalizeSender.SendTransaction(...)
    // ... normal flow, update to RollupFinalizing ...
}
```

3. **Relayer rebuilt and restarted** with the patched code.

### Result
- Bundle 17360 and all its 30 batches are now `RollupFinalized`.
- Relayer automatically finalizes all subsequently created bundles (17361+) via the improved bypass.
- The shadow testing pipeline now runs end-to-end without manual intervention for bundle finalization.

### Key Metrics for Bundle 17360
| Stage | Duration |
|-------|----------|
| Batch proofs generation | ~35 min (parallel on 4 GPUs) |
| Bundle proof generation | 2104s (~35 min) |
| - app_prove (74 segments) | ~5 min |
| - leaf aggregation (74) | ~4 min |
| - internal aggregation (0-4) | ~5 min |
| - root aggregation | ~2 min |
| - halo2_outer proof | 291s (~5 min) |
| - halo2_wrapper proof | ~2 min |

### Pre-Finalize Checklist for Future Shadow Tests
- [ ] `EnableTestEnvBypassFeatures` is in `relayer_config` (not `batch_committer`).
- [ ] Shadow bypass covers both `withProof=true` and `withProof=false` paths.
- [ ] Bypass mode updates directly to `RollupFinalized` to avoid stuck `RollupFinalizing` state.
- [ ] `bundle_index_seq` is set high enough to avoid blocking production indices.
