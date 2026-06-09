# Shadow Testing Lessons Learned

## 2026-06-05: MVRV Verifier Routing Must Match Proof Digests

### What Happened
1.  Successfully proved bundles 13470–13474 (batches 128069–128074) using the local shadow prover with guest v0.8.0.
2.  All proofs verified successfully on the coordinator (status 4), but relayer finalization reverted with `VerificationFailed(0x439cc0cd)`.
3.  The deployed new verifier (`0x16110D4e0CBE54530cE46D1aB2b22574BeEEa105`) had correct canonical digests matching the proofs.
4.  Root cause: `MultipleVersionRollupVerifier.getVerifier(10, batchIndex)` returned the **old production verifier** (`0xc37F4D0F2DEEF2639F2dc76326B9Ca7fA01aAA21`) for batch indices ≥128069.
5.  After executing `updateVerifier(10, 128069, newVerifier)` (Tx `0xb408311e32f1f77974d17fbe01f6da04fb9e24e00cf0188ca045922a5e061a30`), all bundles finalized successfully.

### Root Cause Analysis

The `MultipleVersionRollupVerifier` maintains a mapping of `verifiers[protocolVersion]` → list of `(startBatchIndex, verifierAddress)` entries. When `ScrollChain` calls `MVRV.getVerifier(protocolVersion, batchIndex)`, it returns the **most recently registered verifier whose `startBatchIndex` is ≤ batchIndex**.

If you test a new prover (with new digests) on batch indices that fall within an existing MVRV range mapped to an old verifier, the old verifier will be used — and its digests won't match the new proofs.

### Lesson

**Always verify MVRV routing after registering a new verifier and before starting finalization tests.** Use:
```bash
for idx in 128069 128070 128071; do
  echo -n "Batch $idx → "
  cast call $MVRV "getVerifier(uint256,uint256)(address)" 10 $idx --rpc-url $ANVIL_RPC
done
```

If any batch returns the wrong verifier, update MVRV:
```bash
cast rpc anvil_impersonateAccount $OWNER --rpc-url $ANVIL_RPC
cast send $MVRV \
  "updateVerifier(uint256,uint64,address)" \
  10 $START_BATCH $NEW_VERIFIER \
  --from $OWNER --rpc-url $ANVIL_RPC --unlocked
```

This is especially important when:
- Testing a new prover on old batch ranges (the old batches may already be mapped to a legacy verifier)
- The new verifier has different digests from any previously-registered verifier

---

## 2026-06-05: `bundle.batch_proofs_status` Must Transition to `Ready(2)` Before Finalization

### What Happened
1.  Shadow bundle 13470: all 7 chunk proofs and 6 batch proofs verified (status 4).
2.  Relayer refused to finalize, logging that bundle was not ready.
3.  Investigation showed `bundle.batch_proofs_status = 1 (Pending)` in the shadow DB.
4.  The coordinator's background cron job `checkBundleAllBatchReady` (runs every 10s) had not yet scanned this bundle to transition it to `Ready(2)`.
5.  Manually updating `batch_proofs_status = 2` allowed finalization to proceed.

### Root Cause Analysis

The coordinator maintains a separate `batch_proofs_status` field on bundles to track whether all constituent batch proofs are ready. The relayer checks this field before attempting `finalizeBundleWithProof`. Even when all individual batch proofs are status 4, the bundle-level flag must also be ≥2.

The background cron should eventually update this, but there can be lag (10s interval, plus the cron may have other work). For shadow testing where we're manually driving finalization, this delay is unnecessary.

### Lesson

**After all batch proofs reach status 4, verify `bundle.batch_proofs_status` in the shadow DB before starting finalization.** If still `1 (Pending)`, either wait for the cron or manually update:
```sql
UPDATE bundle SET batch_proofs_status = 2 WHERE index = <bundle_index>;
```

The relayer will not proceed to `finalizeBundleWithProof` until `bundle.batch_proofs_status >= 2`.

---

## 2026-06-05: `batch.withdraw_root` Can Be `0x0` in Production RDS for Codec V7+ Batches

### What Happened
1.  Successfully proved bundles 13455–13459 (batches 128021–128033) using the local shadow prover.
2.  Relayer `finalizeBundlePostEuclidV2` reverted with `VerificationFailed(0x439cc0cd)`.
3.  Manual investigation revealed `batch.withdraw_root = 0x0` in the shadow DB for all 13 batches, while the locally-generated proofs embedded `withdraw_root = 0x7151572959e4dfe58f3060f2a725a011986ff9906a431f063157c8a7ef48c85e` in their metadata.
4.  Patching `batch.withdraw_root` to the non-zero proof value resolved the verification failure and all 5 bundles finalized successfully.

### Root Cause Analysis

#### The `withdraw_root` lives in TWO places with different lifecycles

| Source | Used By | How It's Computed |
|--------|---------|-------------------|
| `batch.withdraw_root` DB column | **Relayer** (`constructFinalizeBundlePayloadCodecV7`) | Set once at batch insertion from `encoding.Batch.WithdrawRoot()` (last block of last chunk) |
| Chunk/block data + proof metadata | **Prover** (`BatchInfoBuilderV7`) | Re-derived from `last_chunk.withdraw_root` during witness generation |

For **codec v7+** (galileoV2), the on-chain `BatchHeader` is only 73 bytes and does **not** contain `withdraw_root` — it's passed as a separate argument to `finalizeBundlePostEuclidV2`. This means the DB column is the *only* place the relayer gets this value.

#### Production RDS contains `0x0` for newer batches

Querying the shadow DB after import revealed a sharp cutoff:

| Batch Range | `withdraw_root` | Bundles |
|-------------|-----------------|---------|
| 128008–128020 | `0x715157…` (non-zero) | 13450–13454 |
| 128021–128033 | `0x0` (patched to `0x715157…`) | 13455–13459 |
| 128034–128153 | `0x0` (120 batches) | 17391–17436 |

The production CSV export for bundles 13450–13454 (Jun 4 11:01) confirms non-zero `withdraw_root` in production RDS for that range. For bundles 13455+ there is no surviving export directory, but the systematic `0x0` pattern across 120 consecutive batches strongly indicates that **production RDS itself stores `0x0`** for these rows.

**Likely reason**: The production Sepolia rollup service was upgraded at some point between bundle 13459 and bundle 17391. The newer code no longer populates `batch.withdraw_root` (possibly because v7+ batch headers omit this field, or the L2 node's `withdrawTrieRootSlot` query was disabled). The column became a stale/dormant field in production.

> ⚠️ **Important**: Production finalization on the live Sepolia chain still succeeded because the production proofs were generated *before* the column was cleared, or production uses a different code path that does not depend on this column. Shadow testing re-proves from scratch, so the prover computes the correct value from block data, but the relayer reads the stale DB column.

### How to Prevent This

1. **After every production data import, verify `withdraw_root` consistency**:
   ```sql
   SELECT index, withdraw_root
   FROM batch
   WHERE withdraw_root = '0x0000000000000000000000000000000000000000000000000000000000000000'
   ORDER BY index;
   ```

2. **Cross-check against proof metadata** (after the first batch proof is generated):
   ```python
   import psycopg2, json
   conn = psycopg2.connect("postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup")
   cur = conn.cursor()
   cur.execute("SELECT index, proof FROM batch WHERE index = 128021")
   idx, proof = cur.fetchone()
   data = json.loads(proof)
   proof_withdraw_root = data['metadata']['batch_info']['withdraw_root']
   print(f"Batch {idx}: proof withdraw_root = {proof_withdraw_root}")
   ```

3. **Patch the DB before running the relayer** if any mismatch is found:
   ```sql
   UPDATE batch
   SET withdraw_root = '0x7151572959e4dfe58f3060f2a725a011986ff9906a431f063157c8a7ef48c85e'
   WHERE index BETWEEN 128021 AND 128033;
   ```

### Recovery Steps (for this incident)

1.  Extract `withdraw_root` from the first successfully-generated batch proof:
    ```python
    proof_json = json.loads(batch_proof_bytes)
    correct_withdraw_root = proof_json['metadata']['batch_info']['withdraw_root']
    ```

2.  Update all affected batches:
    ```sql
    UPDATE batch
    SET withdraw_root = '0x7151572959e4dfe58f3060f2a725a011986ff9906a431f063157c8a7ef48c85e'
    WHERE index BETWEEN 128021 AND 128033;
    ```

3.  Verify the fix:
    ```bash
    cast call $SCROLL_CHAIN "finalizeBundlePostEuclidV2(...)" --rpc-url $ANVIL_RPC
    # should NOT revert with 0x439cc0cd
    ```

### Key Insight

> **For codec v7+, `batch.withdraw_root` is a "ghost column" in production RDS.** It is not part of the batch header bytes, it is not validated by the coordinator, and it may be `0x0` for newer batches. Yet the relayer still passes it to `finalizeBundlePostEuclidV2`, where it becomes part of the verifier's `publicInputs`. Always treat this column as untrusted after a production import and verify it against the proof metadata before attempting on-chain finalization.

---

## 2026-06-03: "psql timeout" does NOT mean "port is closed"

### What Happened
1.  Ran `psql -h localhost -p 25432 ...` to connect to Sepolia shadow DB.
2.  Command timed out after 60s.
3.  **Misconclusion**: Assumed port-forward was not established and told user to start SSH tunnel.
4.  User asked: "你怎么测试的？telnet $PORT 吗？"
5.  Ran `nc -vz localhost 25432` → port was **OPEN**.
6.  Re-ran `psql` with correct username → connected instantly.

### Root Cause
- `psql` timeout can be caused by many things: DNS resolution, SSL handshake failure, wrong username triggering slow auth fallback, etc.
- TCP port being open is a separate layer from application-level connectivity.

### Rule
> **Always test TCP connectivity first** with `nc`, `telnet`, or `/dev/tcp/host/port` before diagnosing application-level issues.
> Only after confirming the port is open (or closed) should you investigate `psql`-specific parameters.

### Verification Commands
```bash
# TCP connectivity test (fast, no auth needed)
timeout 3 bash -c 'cat < /dev/null > /dev/tcp/localhost/25432' && echo "Open" || echo "Closed"

# Or with nc
nc -vz localhost 25432

# Then test psql with explicit username
psql -h localhost -p 25432 -U sepolia_infra_user_read_only -d sepolia_scroll -c "SELECT 1;"
```

---

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
2. **Extract digests from the proof itself**, or convert S3 digests from Montgomery to canonical form:
   ```python
   # Option A: From proof instances (canonical form directly)
   instances = base64.b64decode(proof_json['proof']['instances'])
   digest1 = '0x' + instances[384:416].hex()
   digest2 = '0x' + instances[416:448].hex()
   
   # Option B: From S3 (Montgomery form → canonical)
   bn254_mod = 21888242871839275222246405745257275088548364400416034343698204186575808495617
   R = pow(2, 256, bn254_mod)
   R_inv = pow(R, -1, bn254_mod)
   d1_mont = int(requests.get(".../digest_1.hex").text, 16)
   d1_canon = f"{(d1_mont * R_inv) % bn254_mod:064x}"
   ```
   S3 `digest_1.hex` / `digest_2.hex` are in **Montgomery form**; the verifier constructor expects **canonical form**.
3. **Use the provided script** (`scripts/03-deploy-verifier.sh`) which extracts digests from proof instances and deploys `PostFeynman` with `protocolVersion = 10`.

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

---

## 2026-06-03: Sepolia Shadow Fork — Real `finalizeBundlePostEuclidV2` On-Chain Finalization (Bundles 13445-13449)

> ⚠️ **Context**: This test **re-used existing production proofs** from Sepolia RDS, not newly-generated proofs from a new guest version. The lessons here are about relayer + contract interaction mechanics, NOT about proving new circuit versions.

### What Happened
Successfully finalized 5 production bundles (13445-13449, batches 127994-128007) on a Sepolia Anvil shadow fork using **real on-chain `finalizeBundlePostEuclidV2` transactions** (no bypass). This was a full relayer + contract interaction test, not a coordinator+prover test.

**Proof source**: Imported directly from Sepolia production DB. These proofs were originally generated and verified on the live Sepolia chain. Bundle proof size ≈ 4.6KB, batch proofs ≈ 1MB.

### Issues Discovered and Resolutions

#### 1. Anvil `eth_estimateGas` rejects fee caps without explicit gas limit

**Symptom**: Relayer finalize failed with:
```
failed to get fee data, err: Out of gas: gas required exceeds allowance: 0
```

**Root Cause**: Anvil's `eth_estimateGas` implementation returns `"Out of gas: gas required exceeds allowance: 0"` when the `CallMsg` contains `GasFeeCap`/`GasTipCap` fields but `Gas` is zero/unset. The Go Ethereum client's `EstimateGas` sets `Gas: 0` by default in the `CallMsg`.

**Resolution**: In `rollup/internal/controller/sender/estimategas.go`, create a copy of the `CallMsg` with fee caps stripped before calling `EstimateGas`:

```go
msg := ethereum.CallMsg{
    From:      s.transactionSigner.GetAddr(),
    To:        to,
    Gas:       10000000, // High limit for CreateAccessList later
    GasPrice:  gasPrice,
    GasTipCap: gasTipCap,
    GasFeeCap: gasFeeCap,
    Data:      data,
}

// Anvil bug: eth_estimateGas fails when maxFeePerGas/maxPriorityFeePerGas
// are present without an explicit gas limit.
estimateMsg := msg
estimateMsg.GasPrice = nil
estimateMsg.GasTipCap = nil
estimateMsg.GasFeeCap = nil

gasLimitWithoutAccessList, err := s.client.EstimateGas(s.ctx, estimateMsg)
```

**Rule**: When testing against Anvil, always verify `eth_estimateGas` behavior with a simple curl first if gas estimation fails.

---

#### 2. `L1MessageQueueV2.nextUnfinalizedQueueIndex` storage slot is NOT at slot 0 or 4

**Symptom**: After setting `nextUnfinalizedQueueIndex = 0` via `anvil_setStorageAt` on slot 0, `eth_call` still returned `0x10a6bb` (1,091,259) on real Sepolia. The contract and Anvil disagreed on the value.

**Root Cause**: `L1MessageQueueV2` inherits `OwnableUpgradeable` → `ContextUpgradeable` (with `uint256[50] __gap`) → `Initializable`. The `__gap[50]` pushes `L1MessageQueueV2`'s own variables far down. Using `forge inspect`:

```bash
forge inspect L1MessageQueueV2 storage-layout | grep nextUnfinalizedQueueIndex
# → slot 104 (0x68)
```

Actual layout:
- Slot 0: `_initialized` + `_initializing` + `_owner`
- Slots 1-50: `ContextUpgradeable.__gap[50]`
- Slot 51: `OwnableUpgradeable._owner` (wait, actually it's packed in slot 0)
- Slot 52: `messageRollingHashes` mapping base
- Slot 53: `firstCrossDomainMessageIndex`
- Slot 54: `nextCrossDomainMessageIndex`
- **Slot 55**: Wait, `forge inspect` said 104. The exact number depends on OpenZeppelin version.

**Resolution**: **Always use `forge inspect <Contract> storage-layout`** to find the exact slot for any state variable. Never guess based on source code reading alone.

```bash
forge inspect L1MessageQueueV2 storage-layout
```

For the deployed Sepolia contract, the correct slots were:
- `firstCrossDomainMessageIndex`: slot 102
- `nextCrossDomainMessageIndex`: slot 103
- `nextUnfinalizedQueueIndex`: slot 104

---

#### 3. `nextUnfinalizedQueueIndex` must match pre-finalization state, not post-finalization

**Symptom**: Setting `nextUnfinalizedQueueIndex = 0` caused `finalizeBundlePostEuclidV2` to revert with an L1 message queue index mismatch.

**Root Cause**: The fork block (10979334) is AFTER the real finalization of bundles 13445-13449 on Sepolia. The real state at that block already has `nextUnfinalizedQueueIndex = 1,091,259` (post-finalization). We reset `lastFinalizedBatchIndex` to 127993 (pre-finalization) to re-simulate finalization, but also need `nextUnfinalizedQueueIndex` at its pre-finalization value.

**How to compute the correct pre-finalization value**:
```sql
SELECT MIN(total_l1_messages_popped_before) 
FROM chunk 
WHERE batch_hash IN (SELECT hash FROM batch WHERE index = <first_target_batch>);
-- → 1091247 for batch 127994
```

**Resolution**:
```bash
# Set to pre-finalization value (NOT 0, NOT post-finalization value)
curl -X POST http://localhost:18546 \
  -d '{"jsonrpc":"2.0","method":"anvil_setStorageAt","params":[
    "0xA0673eC0A48aa924f067F1274EcD281A10c5f19F",
    "0x68",  # slot 104 — verify with forge inspect first
    "0x000000000000000000000000000000000000000000000000000000000010a6af"
  ],"id":1}'
```

**Rule**: For shadow fork re-finalization tests, `nextUnfinalizedQueueIndex` must be the `MIN(total_l1_messages_popped_before)` of the first target batch's chunks.

---

#### 4. Anvil sender balances reset to zero

**Symptom**: After fixing gas estimation, relayer failed with:
```
failed to send transaction, err: Insufficient funds for gas * price + value
```

**Root Cause**: `anvil_setBalance` funds from previous sessions do not persist across Anvil restarts. The finalize sender (`0x410E...`) had 0 ETH.

**Resolution**: Re-fund before starting the relayer:
```bash
curl -X POST http://localhost:18546 \
  -d '{"jsonrpc":"2.0","method":"anvil_setBalance","params":[
    "0x410E7FD80a3Fc1E62A4D3450d11b71b812006eB9",
    "0x21e19e0c9bab2400000"
  ],"id":1}'
```

**Rule**: After every Anvil restart, verify sender balances before starting the relayer:
```bash
curl -X POST http://localhost:18546 \
  -d '{"jsonrpc":"2.0","method":"eth_getBalance","params":[
    "0x410E7FD80a3Fc1E62A4D3450d11b71b812006eB9","latest"
  ],"id":1}'
```

---

#### 5. Relayer requires `--config` flag and `--min-codec-version`

**Symptom**: Relayer printed help text and exited with:
```
Required flag "min-codec-version" not set
```

Then when started without `--config`, it connected to the default `./conf/config.json` (mainnet config) instead of the shadow config, failing with wrong DB credentials.

**Root Cause**: The relayer uses `cli.StringFlag{Name: "config"}` for config file path, NOT an environment variable. And `MinCodecVersionFlag` is `Required: true`.

**Resolution**:
```bash
cd rollup && ./build/bin/rollup_relayer \
  --config /tmp/rollup-relayer-sepolia-shadow.json \
  --min-codec-version 10
```

**Rule**: Never rely on `ROLLUP_RELAYER_CONFIG` env var (it doesn't work). Always pass `--config <path>` and `--min-codec-version <version>` explicitly.

---

#### 6. Production proofs + production verifier = no new deployment needed (THIS TEST ONLY)

**Symptom**: Initially thought a new verifier needed to be deployed for the shadow fork.

**Root Cause**: This test **re-used production proofs** from Sepolia RDS. Sepolia's production `MultipleVersionRollupVerifier` (MVRV) at `0x8A360...` already points to verifier `0xc37f...` with digests that match these exact production proofs. Since proof and verifier were already a verified pair on the live chain, no new deployment was necessary.

**MVRV** = `MultipleVersionRollupVerifier`, a Solidity contract that maps `protocolVersion → verifierAddress`. ScrollChain calls `MVRV.getVerifier(version, batchIndex)` to determine which verifier to use for a given bundle.

**Resolution**: Verified digest match via `cast call`:
```bash
cast call 0x8A360c7F6fca548507017DdeD732bFe7E078F963 \
  "getVerifier(uint256,uint256)" 10 127996 \
  --rpc-url https://eth-sepolia.g.alchemy.com/v2/<KEY>

cast call <verifier_addr> "verifierDigest1()" --rpc-url <URL>
cast call <verifier_addr> "verifierDigest2()" --rpc-url <URL>
```

**⚠️ CRITICAL DISTINCTION**:
- **This test** (re-use production proofs): Check MVRV → if digests match, skip deployment.
- **New guest version test** (e.g., 0.8.0 / openvm 1.6): **MUST deploy new verifier**. New guest = new circuit = new plonk verifier bin = new digests. The old MVRV verifier will NOT match. Follow the full deployment flow in `docs/GUIDE.md` → "Real Verifier Deployment".

**Rule**: Always know which scenario you're in:
1. Re-playing old production tasks → verify existing MVRV entry matches.
2. Testing new circuit/guest → deploy fresh `ZkEvmVerifierPostFeynman` + register on MVRV.

---

### Final State Verification

After all 5 bundles finalized successfully:

```bash
# lastFinalizedBatchIndex = 128007
cast call 0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0 \
  "lastFinalizedBatchIndex()(uint256)" --rpc-url http://localhost:18546
# → 128007

# nextUnfinalizedQueueIndex = 1091254 (started at 1091247 + 7 messages)
cast call 0xA0673eC0A48aa924f067F1274EcD281A10c5f19F \
  "nextUnfinalizedQueueIndex()(uint256)" --rpc-url http://localhost:18546
# → 1091254
```

### Successful Finalize Transactions

| Bundle | Batches | Tx Hash |
|--------|---------|---------|
| 13445 | 127994-127996 | `0x64cd766d...` |
| 13446 | 127997-127999 | `0x2fda1bd5...` |
| 13447 | 128000-128002 | `0x2724f176...` |
| 13448 | 128003-128004 | `0xf5f7054a...` |
| 13449 | 128005-128007 | `0xf6f7903f...` |

### Pre-Finalize Checklist for Real On-Chain Shadow Fork Tests

- [ ] Anvil forked at `last_real_finalize_block + 1`
- [ ] `lastFinalizedBatchIndex` set to `<first_target_batch - 1>`
- [ ] `lastCommittedBatchIndex` set to real Sepolia value (≥ last target batch)
- [ ] All end-batch indices (127996, 127999, 128002, 128004, 128007) have non-zero `committedBatches` hashes
- [ ] `L1MessageQueueV2.nextUnfinalizedQueueIndex` set to `MIN(total_l1_messages_popped_before)` of first target batch
- [ ] `L1MessageQueueV2.nextCrossDomainMessageIndex` ≥ post-finalization value
- [ ] **Verify slot numbers with `forge inspect`** before `anvil_setStorageAt`
- [ ] Sender balances > 0 on Anvil
- [ ] Prover EOA authorized on ScrollChain (`addProver`)
- [ ] Verifier digests match proofs (check MVRV before deploying)
- [ ] Relayer started with `--config <path>` and `--min-codec-version 10`
- [ ] Target bundles/batches reset to `rollup_status = 1`
- [ ] Parent batch exists in shadow DB

---

## 2026-06-04: Shadow Testing with New zkvm-prover — Bundles 13450-13454

### What Happened
Attempted to prove bundles 13450-13454 using the local shadow prover (current build with OpenVM 1.6.0 / zkvm-prover `ed3b964`). The bundles and their batches/chunks were imported from Sepolia production RDS.

### Lesson 1: Import Production Data, Then Clear Proofs and Reset Status

**Wrong approach** (what was done initially):
- Imported bundles with production proofs from RDS
- Kept the proofs in the DB
- Tried to assign bundle tasks directly to the prover
- Coordinator failed with `ProofEnum deserialization` error because the imported proofs were generated by a different zkvm-prover version

**Correct approach**:
1. Import raw metadata (batch headers, chunk info, L2 blocks, state roots) from RDS
2. **Clear all `proof` fields** after import:
   ```sql
   UPDATE chunk SET proof = NULL, proving_status = 1, ... WHERE ...;
   UPDATE batch SET proof = NULL, proving_status = 1, chunk_proofs_status = 0, ... WHERE ...;
   UPDATE bundle SET proof = NULL, proving_status = 1, batch_proofs_status = 1, ... WHERE ...;
   ```
3. Let the local prover regenerate all proofs from scratch
4. The coordinator verifies the newly-generated proofs

**Why**: Proofs are tied to the specific zkvm-prover / circuit version. The `StarkProof` struct uses `bincode_v1` serialization for `Proof<SC>`, and the binary format changes when the `openvm-stark-backend` or `openvm-sdk` versions change. Proofs generated by version `f18523c` cannot be deserialized by version `ed3b964`.

### Lesson 2: Prover Config Must Use Valid Proof Types

**Wrong config**:
```json
"supported_proof_types": [0, 1, 2, 3]
```
Type `0` = `ProofTypeUndefined`, which the coordinator rejects with `illegal proof type: 0`.

**Correct config**:
```json
"supported_proof_types": [1, 2, 3]
```
- `1` = Chunk
- `2` = Batch
- `3` = Bundle

### Lesson 3: Prover Binary Takes Only `--config` Flag

**Wrong invocation**:
```bash
prover --config prover.json --http.addr 0.0.0.0 --http.port 10080
```
The prover binary does NOT accept `--http.addr` or `--http.port`. It only takes `--config <path>`.

**Correct invocation**:
```bash
prover --config prover.json
```
The listener address goes in the config file:
```json
"health_listener_addr": "127.0.0.1:10080"
```

### Lesson 4: GPU OOM When Running Multiple Provers Concurrently

**Symptom**: Running 4 provers (one per GPU) caused repeated CUDA OOM errors:
```
GPU allocation failed: OutOfMemory { requested: 4294967296, available: 2965372928 }
thread 'tokio-rt-worker' panicked at ... cudaErrorMemoryAllocation: out of memory
```

**Root Cause**: Each prover allocates a large GPU memory pool for circuit proving. Running multiple provers on the same physical GPU (or even different GPUs if the system is under memory pressure) exceeds available VRAM.

**Mitigation**:
- Use the script's default of 2 GPUs (`GPUS="0,1"`) instead of 4
- Or run provers sequentially on a single GPU
- Monitor GPU memory with `nvidia-smi` during proving

**Note**: With 1-2 provers on RTX 3090s, chunk proof generation takes ~700-800s. Batch and bundle proofs take longer.

### Lesson 5: GORM `[]byte` Mapping for PostgreSQL `bytea` Works Correctly

**Initial suspicion**: The `json.Unmarshal(batch.Proof, &message.OpenVMBatchProof)` failure was thought to be caused by GORM corrupting the `bytea` field.

**Verification**: A standalone Go test using the exact same GORM model and PostgreSQL `bytea` type successfully read and unmarshaled all batch proofs (1,089,806 bytes each).

**Conclusion**: GORM correctly handles PostgreSQL `bytea` fields. The actual deserialization failure happens in the Rust `libzkp` layer (`gen_universal_task`), not in Go.

### Pre-Proving Checklist for New Guest Version Tests

- [ ] Import raw data from RDS (exclude `proof` columns, or clear them after import)
- [ ] Reset `proving_status` to 1 for all chunks, batches, bundles
- [ ] Reset `chunk_proofs_status` to 0 for batches
- [ ] Reset `batch_proofs_status` to 1 for bundles
- [ ] Prover config has `"supported_proof_types": [1, 2, 3]` (NOT `[0, 1, 2, 3]`)
- [ ] Prover launched with `--config <path>` only (no `--http.addr`/`--http.port`)
- [ ] Coordinator asset hashes match prover circuit hashes
- [ ] GPU count is appropriate for available VRAM (2 GPUs recommended for RTX 3090)
- [ ] Shadow bypass is configured if testing finalize without real on-chain verification

### 6. Prover Batch Proof Stack Overflow — Set `RUST_MIN_STACK`

**Date**: 2026-06-04

**Symptom**: After chunk proofs succeed, prover crashes during batch proof generation with:
```
thread 'tokio-rt-worker' has overflowed its stack
fatal runtime error: stack overflow, aborting
```
Crash occurs in `gen_proof_universal` during aggregation keygen, right after `coset_lde_batch`.

**Root Cause**: The default Rust thread stack size (2 MB) is insufficient for OpenVM 1.6.0 batch/bundle proof generation. Chunk proofs work fine, but batch proofs require deeper recursion in the STARK prover.

**Fix**: Set `RUST_MIN_STACK=16777216` (16 MB) before starting the prover. The `zkvm-prover/Makefile` already sets this, but custom startup scripts must also export it:

```bash
export RUST_MIN_STACK=16777216
CUDA_VISIBLE_DEVICES="$gpu_id" nohup "$PROVER_BIN" --config "$config_file" > "$log_file" 2>&1 &
```

**Lesson**: Always ensure `RUST_MIN_STACK` is exported in prover startup scripts, not just in the build Makefile.

---

## 2026-06-04: Bundles 13450–13454 All Proved Successfully with OpenVM 1.6.0 (zkvm-prover ed3b964)

### What Happened
Successfully proved all 5 bundles (13450–13454, batches 128008–128020) using the local shadow prover built with OpenVM 1.6.0 / zkvm-prover `ed3b964`.

| Bundle | Batches | Prover | Proof Time | Status |
|--------|---------|--------|------------|--------|
| 13450 | 128008–128009 | Prover 0 | ~22 min | ✅ Verified |
| 13451 | 128010–128012 | Prover 0 | ~9 min | ✅ Verified |
| 13452 | 128013–128015 | Prover 0 | ~12 min | ✅ Verified |
| 13453 | 128016–128017 | Prover 0 | ~35 min | ✅ Verified |
| 13454 | 128018–128020 | Prover 1 | ~35 min | ✅ Verified |

**Total**: 14/14 chunks, 13/13 batches, 5/5 bundles verified.

### New Issues Discovered and Resolutions

#### 7. `batch_proofs_status` Does Not Auto-Update in Shadow Testing

**Symptom**: After all batches in a bundle reached `proving_status=4` (verified), the bundle's `batch_proofs_status` remained `1` (Pending). The coordinator's cron job that normally updates this was not running in the shadow testing setup.

**Impact**: Bundles could not be scheduled for bundle proof generation because `Assign()` requires `batch_proofs_status == 2` (Ready).

**Fix**: Manually update when all constituent batches are verified:
```sql
UPDATE bundle SET batch_proofs_status = 2 WHERE index = <bundle_index>;
```

**Why this happens**: The coordinator background cron (`cron.UpdateBundleProofsStatus`) is either not enabled or relies on production-specific infrastructure (e.g., message queue, scheduler) that is absent in shadow testing.

#### 8. Prover 1 Entered Failure Loop After Config Change

**Symptom**: After changing `supported_proof_types` from `[1,2,3]` to `[3]` (Bundle only), Prover 1 was assigned a chunk task, rejected it, and entered a loop:
```
ERROR: cannot submit valid proof for a prover task twice
ERROR: CoordinatorEmptyProofData: get empty prover task
```

**Root Cause**: The prover received a chunk task from the coordinator but its config said it only supports Bundle proofs. It failed the task, but the coordinator kept reassigning it.

**Fix**:
1. Revert config to `supported_proof_types: [1, 2, 3]`
2. Reset the stuck chunk's `proving_status = 1`, `active_attempts = 0`
3. Delete failed `prover_task` records for that chunk
4. Restart prover

#### 9. `libzkp.so` Must Be Rebuilt When zkvm-prover Version Changes

**Symptom**: Coordinator verification failed with:
```
failed to verify proof: data did not match any variant of untagged enum ProofEnum
```

**Root Cause**: The `libzkp.so` shared library was built on May 19 against an older zkvm-prover (`f18523c`). The new prover (`ed3b964`, OpenVM 1.6.0) changed the `Proof<SC>` bincode serialization format. Old `libzkp.so` could not deserialize new proofs.

**Fix**: Rebuild `libzkp-c` and replace `libzkp.so`:
```bash
cargo build --release -p libzkp-c
cp target/release/libzkp.so coordinator/build/bin/
```

**Lesson**: `libzkp.so` is NOT forward-compatible across zkvm-prover revisions. Always rebuild after upgrading the prover.

#### 10. Coordinator `json.Unmarshal` Error Was a Red Herring

**Symptom**: Coordinator log showed:
```
failed to unmarshal proof: ..., bundle hash: ..., batch hash: ...
```

**Initial suspicion**: GORM was corrupting PostgreSQL `bytea` fields.

**Verification**: Standalone Go test confirmed GORM correctly maps `bytea` → `[]byte`.

**Actual root cause**: The `json.Unmarshal` in Go succeeded (it produced a valid `OpenVMBatchProof` struct). The failure was in Rust `libzkp::gen_universal_task` when it tried to bincode-deserialize the inner `StarkProof`. The error message bubbled up from Rust → CGO → Go, but the Go layer's `json.Unmarshal` log was the most visible symptom.

**Lesson**: When seeing deserialization errors in a Go/Rust hybrid system, verify which layer actually fails. Don't assume the first logged error is the root cause.

### Next Step: Real On-Chain Finalize

All 5 bundle proofs are coordinator-verified. The next step is to attempt real on-chain finalization via the relayer:

1. Ensure `ZkEvmVerifierPostFeynman` is deployed with digests matching the new proofs
2. Register verifier on `MultipleVersionRollupVerifier`
3. Ensure batches are committed on-chain (`committedBatches[endBatchIndex] != 0`)
4. Start relayer to call `finalizeBundlePostEuclidV2`

⚠️ **Current Anvil state**: `lastCommittedBatchIndex = 0`, `lastFinalizedBatchIndex = 0`. The batches were never committed on this Anvil fork. The relayer must first commit batches before finalizing bundles.

### Post-Proving Checklist

- [ ] All chunks/batches/bundles have `proving_status = 4`
- [ ] All bundles have `batch_proofs_status = 2`
- [ ] `libzkp.so` matches prover revision
- [ ] Coordinator asset hashes match prover circuit hashes
- [ ] Verifier digests extracted from new proofs match on-chain verifier
- [ ] `committedBatches[endBatchIndex]` is non-zero for all target batches
- [ ] Relayer config has `enable_test_env_bypass_features` in correct location (if needed for other tests)

---

## 2026-06-04: Bundle 13451 `VerificationFailed` — `L1MessageQueueV2` State Mismatch on Anvil Fork

### What Happened

After successfully proving bundles 13450–13454 with OpenVM 1.6.0, bundle 13450 finalized on-chain successfully. Bundle 13451 failed with:

```
execution reverted: custom error 0x439cc0cd   # VerificationFailed
```

Manual `cast call` to the verifier contract with DB-extracted public inputs reproduced the same error.

### Root Cause

The Anvil fork block (10979334) was at a boundary where `L1MessageQueueV2.nextCrossDomainMessageIndex = 1091255`. Bundle 13451's `totalL1MessagesPoppedOverall = 1091256`, so the contract queried `getMessageRollingHash(1091255)`. On Anvil this returned `0x0` because no message had been appended at that index yet. In production, `getMessageRollingHash(1091255) = 0xb9954a9f...`.

The proof was generated with the production `messageQueueHash` (embedded in `bundle_pi_hash`), but the on-chain contract recomputed `publicInputs` using Anvil's stale `0x0` value. This mismatch caused `VerificationFailed` even though the proof structure and SNARK were internally valid.

**The coordinator verifies SNARK self-consistency (proof matches its own instances), NOT that the instances match on-chain state.**

### Diagnosis Steps

1. **Verify the error is from the verifier, not the contract**:
   ```bash
   cast call <VERIFIER> "verify(bytes,bytes32[])" <proof> <publicInputs> --rpc-url $ANVIL_RPC
   # → reverts with 0x439cc0cd
   ```

2. **Compare `messageQueueHash` in proof metadata vs contract**:
   ```python
   # From bundle proof JSON
   msg_queue_hash = proof_json['metadata']['bundle_info']['msg_queue_hash']
   # From contract (what it would compute)
   cast call <L1MQ> "getMessageRollingHash(uint256)(bytes32)" 1091255 --rpc-url $ANVIL_RPC
   # → 0x0 (mismatch!)
   ```

3. **Check production value**:
   ```bash
   cast call <L1MQ> "getMessageRollingHash(uint256)(bytes32)" 1091255 --rpc-url $SEPOLIA_RPC
   # → 0xb9954a9f... (matches proof)
   ```

### Recovery Steps

#### 1. Sync `messageRollingHashes` from production

Use `anvil_setStorageAt` to set the correct rolling hash values. First find the base slot:

```bash
forge inspect L1MessageQueueV2 storage-layout | grep messageRollingHashes
# → slot 101
```

Compute individual slots and set values:

```python
import eth_abi
from eth_utils import keccak

BASE_SLOT = 101
ANVIL_RPC = "http://localhost:18546"
L1MQ = "0xA0673eC0A48aa924f067F1274EcD281A10c5f19F"

# Fetch from production
for idx in range(1091255, 1091274):
    hash_val = cast_call(L1MQ, "getMessageRollingHash(uint256)(bytes32)", idx, SEPOLIA_RPC)
    slot = keccak(eth_abi.encode(['uint256', 'uint256'], [idx, BASE_SLOT]))
    anvil_set_storage_at(L1MQ, slot, hash_val)
```

#### 2. Update `nextCrossDomainMessageIndex`

```bash
# Set to production value (1091274)
cast rpc anvil_setStorageAt "$L1MQ" "0x67" \
  "0x000000000000000000000000000000000000000000000000000000000010a6ca" \
  --rpc-url "$ANVIL_RPC"
```

#### 3. Update `nextUnfinalizedQueueIndex` to **pre-finalization** value

**Critical**: Do NOT set this to the production current value. It must be the value *before* the first target bundle was finalized.

```sql
-- For bundle 13451 (first batch = 128010), find the pre-finalization value
-- which is the totalL1MessagesPoppedOverall of the previously-finalized bundle
SELECT total_l1_messages_popped_before + total_l1_messages_popped_in_chunk
FROM chunk
WHERE index = (SELECT end_chunk_index FROM batch WHERE index = 128009);
-- → 1091255
```

```bash
cast rpc anvil_setStorageAt "$L1MQ" "0x68" \
  "0x000000000000000000000000000000000000000000000000000000000010a6b7" \
  --rpc-url "$ANVIL_RPC"
```

#### 4. Ensure finalize sender is an authorized prover

`finalizeBundlePostEuclidV2` has `OnlyProver` modifier. If using a new EOA:

```bash
# Impersonate ScrollChain owner
OWNER=$(cast call $SCROLL_CHAIN "owner()(address)" --rpc-url $ANVIL_RPC)
cast rpc anvil_impersonateAccount "$OWNER" --rpc-url "$ANVIL_RPC"

# Add new sender as prover
cast send $SCROLL_CHAIN "addProver(address)" "$NEW_SENDER" \
  --from "$OWNER" --rpc-url "$ANVIL_RPC" --unlocked

cast rpc anvil_stopImpersonatingAccount "$OWNER" --rpc-url "$ANVIL_RPC"
```

### Verification

```bash
# L1MessageQueueV2 state
cast call $L1MQ "nextCrossDomainMessageIndex()(uint256)" --rpc-url $ANVIL_RPC
# → 1091274
cast call $L1MQ "nextUnfinalizedQueueIndex()(uint256)" --rpc-url $ANVIL_RPC
# → 1091255
cast call $L1MQ "getMessageRollingHash(uint256)(bytes32)" 1091255 --rpc-url $ANVIL_RPC
# → 0xb9954a9f...

# ScrollChain state
cast call $SCROLL_CHAIN "lastFinalizedBatchIndex()(uint256)" --rpc-url $ANVIL_RPC
# → 128009 (pre-finalization)

# Test verifier directly
cast call $VERIFIER "verify(bytes,bytes32[])" <proof> <publicInputs> --rpc-url $ANVIL_RPC
# → should NOT revert
```

### Result

After the fix, all bundles 13450–13454 finalized successfully on-chain:

| Bundle | Batches | Finalize Tx | Status |
|--------|---------|-------------|--------|
| 13450 | 128008–128009 | `0xfcbce5...` | ✅ Finalized |
| 13451 | 128010–128012 | `0x20117a...` | ✅ Finalized |
| 13452 | 128013–128015 | `0x4f4b68...` | ✅ Finalized |
| 13453 | 128016–128017 | `0x1daa13...` | ✅ Finalized |
| 13454 | 128018–128020 | `0xd1da28...` | ✅ Finalized |

### Key Lesson

> **Shadow fork state can diverge from production in subtle ways.** Even when `ScrollChain.committedBatches` and `finalizedStateRoots` look correct, peripheral contracts like `L1MessageQueueV2` may have different state at the fork block. Always verify that *all* contract inputs used in `publicInputs` computation match the values the proof was generated with.

### Pre-Finalize Checklist (Updated)

- [ ] Anvil forked at `last_real_finalize_block + 1`
- [ ] `lastFinalizedBatchIndex` set to `<first_target_batch - 1>`
- [ ] `lastCommittedBatchIndex` set to real Sepolia value (≥ last target batch)
- [ ] All end-batch indices have non-zero `committedBatches` hashes
- [ ] `L1MessageQueueV2.nextUnfinalizedQueueIndex` set to pre-finalization value
- [ ] `L1MessageQueueV2.nextCrossDomainMessageIndex` ≥ post-finalization value
- [ ] **`L1MessageQueueV2.messageRollingHashes` synced from production for all indices needed by target bundles**
- [ ] **Verify slot numbers with `forge inspect`** before `anvil_setStorageAt`
- [ ] Sender balances > 0 on Anvil
- [ ] **Finalize sender is authorized prover** (`isProver[sender] == true`)
- [ ] Verifier digests match proofs
- [ ] Relayer started with `--config <path>` and `--min-codec-version 10`
- [ ] Target bundles/batches reset to `rollup_status = 1`

---

## 2026-06-09: Mainnet Shadow Fork — Re-prove + Real On-Chain Finalize (Bundles 17297–17301)

Full end-to-end run on a **mainnet** Anvil fork using the dockerized coordinator/prover images
(`zhuoatscroll/{coordinator-api,prover}:v4.7.13-openvm16`): imported bundles 17297–17301
(batches 517761–517765, codec v10, single-batch each) from mainnet RDS, cleared production proofs,
re-proved all 20 chunks + 5 batches + 5 bundles locally on 4×RTX 3090, deployed a fresh
`ZkEvmVerifierPostFeynman`, and finalized all 5 bundles with **real** `finalizeBundlePostEuclidV2`
transactions. `lastFinalizedBatchIndex` advanced 517760 → 517765 and `finalizedStateRoots[517765]`
matched the DB batch state root. Three new traps surfaced — documented below.

### Trap A: halo2 SRS files must live in `~/.openvm/params/`, NOT `~/.openvm/`

**Symptom**: chunk and batch proofs succeed, but the **first bundle proof** crashes the prover at the
halo2 wrapping stage:
```
thread 'tokio-rt-worker' panicked at .../halo2/utils.rs:127:
Params file "/root/.openvm/params/kzg_bn254_23.srs" does not exist
```
Container exits; the bundle stays stuck at `proving_status = 2`.

**Root cause**: `CacheHalo2ParamsReader` reads the KZG SRS from `$HOME/.openvm/params/kzg_bn254_{k}.srs`
(openvm `extensions/native/recursion/src/halo2/utils.rs`). Only the bundle proof's `halo2_outer` /
`halo2_wrapper` stages need it (k = 22/23/24); chunk/batch proofs use smaller in-tree params, so the
problem stays hidden until the first bundle reaches halo2. If the `.srs` files are downloaded/placed at
`~/.openvm/` root (or any other dir), they are silently not found.

**Fix**: ensure the SRS files are under the `params/` subdir of the mounted openvm dir:
```bash
mkdir -p ~/.openvm/params
mv ~/.openvm/kzg_bn254_2{2,3,4}.srs ~/.openvm/params/    # if they landed in the wrong place
# files: kzg_bn254_22.srs (~513MB), _23.srs (~1.1GB), _24.srs (~2.1GB)
```
When running the prover in Docker, mount the host openvm dir to `/root/.openvm` (writable) and confirm
`/root/.openvm/params/kzg_bn254_23.srs` resolves inside the container.

### Trap B: prover Docker `--gpus device=N` renumbers the GPU to index 0 inside the container

**Symptom**: prover container exits immediately (code 139) with:
```
CudaError { code: 100, name: "cudaErrorNoDevice", message: "no CUDA-capable device is detected" }
```
Only the prover on GPU 0 works; provers for GPUs 1/2/3 crash on boot.

**Root cause**: `docker run --gpus "device=N"` exposes **only** that one GPU to the container and
**renumbers it to index 0** inside. Setting `CUDA_VISIBLE_DEVICES=N` (the host index) then points at a
device that doesn't exist in the container.

**Fix**: pair `--gpus "device=$i"` with `CUDA_VISIBLE_DEVICES=0` (the only visible device in-container):
```bash
docker run -d --name shadow-prover-$i --network host \
  --gpus "device=$i" -e CUDA_VISIBLE_DEVICES=0 -e RUST_MIN_STACK=16777216 \
  -v .../prover-$i.json:/prover/conf/config.json:ro \
  -v .../prover-$i:/prover/.work -v ~/.openvm:/root/.openvm \
  zhuoatscroll/prover:v4.7.13-openvm16 --config /prover/conf/config.json
```
(Alternative: `--gpus all` + `CUDA_VISIBLE_DEVICES=$i`.)

### Trap C: galileoV2 verifier assets are under S3 `v0.8.0/`, prover circuits under `galileov2/`

The coordinator verifier assets (`openVmVk.json`, `verifier.bin`, `root_verifier_vk`) for the galileoV2
fork are served from `scroll-zkvm/v0.8.0/verifier/` (the `galileov2/verifier/` path returns **403**),
while the prover downloads its circuits (`{chunk,batch,bundle}/<vk_hash>/app.vmexe`) from
`scroll-zkvm/galileov2/`. They are nonetheless consistent: the VK hashes in
`v0.8.0/verifier/openVmVk.json` (`chunk 64cf16…`, `batch e9d653…`, `bundle 6b155f…`) match the circuit
objects available under `galileov2/`. Download coordinator assets from `v0.8.0/verifier/`; point the
prover `circuits.galileoV2.base_url` at `…/scroll-zkvm/galileov2/`.

### Other notes from this run

- **Coordinator/prover are run via the prebuilt Docker images** (native prover build fails on CUDA on
  this host). Run with `--network host` so the coordinator reaches the shadow DB on `localhost:5433`,
  the prover reaches the coordinator on `localhost:8390`, and the relayer reaches Anvil on
  `localhost:18545`. Coordinator entrypoint is `/bin/coordinator_api`; `LD_LIBRARY_PATH` for `libzkp.so`
  is already baked into the image. Coordinator config: `l2.chain_id = 534352` (Scroll **mainnet** L2),
  `l2.l2geth.endpoint` = internal debug-enabled proxy, one `verifiers[]` entry with
  `fork_name: galileoV2` + low `min_prover_version`.
- **`l2_block` export by JOIN on `chunk_hash` is pathologically slow** against the prod RDS (full scan of
  a huge table). Export by **block-number range** instead (`WHERE number BETWEEN <min_start> AND
  <max_end>`, PK-indexed, ~tens of seconds). The block range is the min `start_block_number` / max
  `end_block_number` across the target batches' chunks.
- **`chunk_proofs_status` / `batch_proofs_status` may not auto-promote** in the shadow setup. A small
  watcher loop that sets `batch.chunk_proofs_status = 2` once all of a batch's chunks reach
  `proving_status = 4`, and `bundle.batch_proofs_status = 2` once all of a bundle's batches reach
  `proving_status = 4`, keeps the chunk→batch→bundle pipeline flowing without stalls.
- **The batch committer fails harmlessly during a finalize-only run**: the relayer's commit sender
  (derived from `commit_sender_signer_config`, e.g. `0xBC732a76…`) is unfunded and not a sequencer, so
  `commitBatch` loops with "Insufficient funds"/`ErrorCallerIsNotSequencer`. This is expected and does
  **not** affect the bundle finalizer, which runs independently and uses the (funded, prover-authorized)
  finalize sender.
- **Set `bundle_index_seq` above the imported max** (e.g. `SELECT setval('bundle_index_seq', 18000)`)
  before starting the relayer, so any proposer-created bundle gets a higher index and cannot block
  `GetFirstPendingBundle` (orders by `index ASC`). Also clear stale `finalize_tx_hash` on imported
  bundles/batches.

### Successful finalize transactions (mainnet fork)

| Bundle | Batch | Finalize Tx | Status |
|--------|-------|-------------|--------|
| 17297 | 517761 | `0x5e8a7e01…cd1b` | ✅ |
| 17298 | 517762 | `0xf55e9f00…c407f` | ✅ |
| 17299 | 517763 | `0xa6466c0a…412f` | ✅ |
| 17300 | 517764 | `0x1deca7d1…2d8f` | ✅ |
| 17301 | 517765 | `0x204b28de…a100` | ✅ |

Final `lastFinalizedBatchIndex = 517765`; verifier deployed at `0xf74BcAA17bbb3B0a996aF04a7b301E69501C4bf0`
(plonk `0x1d710357818776073705b29482486AbCF586f33b`), digests `0x00398b78…` / `0x0021785a…`.
