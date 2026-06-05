# Troubleshooting & Pitfalls

> **Read this file first** before starting any shadow fork or shadow coordinator test.
> This directory contains hard-won knowledge from multiple debugging sessions. Blind experimentation will repeat documented mistakes.

## Pre-Flight Ritual (Mandatory)

Before executing a single command:

1. [ ] **Read root `AGENTS.md`** (this file) — refresh the trap list.
2. [ ] **Read `docs/LESSONS_LEARNED.md`** — check if your planned task matches any documented failure mode.
3. [ ] **Read `docs/GUIDE.md`** — verify the specific section matching your task (e.g., "Real Verifier Deployment", "Multi-Bundle Relayer Finalize Test").
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
- **Cause B**: Used S3 `digest_1.hex` / `digest_2.hex` **directly** without Montgomery → Canonical conversion.
- **Rule**: For guest v0.8.0+ proofs, **always use `PostFeynman`** with **canonical-form digests**.
- **Verification**: Extract canonical digests from proof instances:
  ```python
  instances = base64.b64decode(proof_json['proof']['instances'])
  digest1 = '0x' + instances[384:416].hex()   # canonical, offset 384-416
  digest2 = '0x' + instances[416:448].hex()   # canonical, offset 416-448
  ```
  Then deploy with `protocolVersion = 10`.
- **S3 Digests**: If using S3 `digest_1.hex`, convert from Montgomery to canonical first:
  ```python
  bn254_mod = 21888242871839275222246405745257275088548364400416034343698204186575808495617
  R_inv = pow(pow(2, 256, bn254_mod), -1, bn254_mod)
  canonical = (int(s3_hex, 16) * R_inv) % bn254_mod
  ```

### Trap 2: Anvil Forks Wrong Chain
- **Symptom**: `ScrollChain` proxy has no code, or `eth_chainId` returns `534352`.
- **Cause**: Anvil pointed at Scroll L2 RPC instead of Ethereum L1 RPC.
- **Rule**: Anvil must fork **Ethereum L1** (`chainId=1`). The ScrollChain proxy lives on L1.

### Trap 3: L2 RPC Missing `debug_executionWitness`
- **Symptom**: Coordinator panics at startup or chunks never get assigned.
- **Cause**: Public RPC (`mainnet-rpc.scroll.io`, `sepolia-rpc.scroll.io`) blocks debug methods.
- **Rule**: Use **internal** L2 RPC proxies only.

### Trap 4: `committedBatches` Sparse (EuclidV2)
- **Symptom**: `ErrorIncorrectBatchHash(0x2a1c1442)`.
- **Cause**: EuclidV2 only stores the **last batch hash** of each commit tx. Intermediate batches have `committedBatches[index] = 0x0`.
- **Rule**: The contract checks `committedBatches[batchIndex]` where `batchIndex` is the **end batch** of the bundle. Verify this is non-zero before finalizing.

### Trap 5: `L1MessageQueueV2` Index Mismatch (Sepolia)
- **Symptom**: `ErrorFinalizedIndexTooLarge(0x16465978)` or `ErrorFinalizedIndexTooSmall`.
- **Cause**: `nextUnfinalizedQueueIndex` does not match the pre-finalization state expected by the first target batch. The fork block is AFTER real finalization, so the real state has post-finalization values.
- **Rule**: 
  1. Set `nextUnfinalizedQueueIndex` to `MIN(total_l1_messages_popped_before)` of the first target batch's chunks (from DB).
  2. **Use `forge inspect L1MessageQueueV2 storage-layout`** to find the exact storage slot (it's slot 104, NOT slot 0 or 4, due to OpenZeppelin `__gap`).
  3. Never guess storage slots from source code.

### Trap 6: Parent Batch Missing
- **Symptom**: Relayer logs `Batch.GetBatchByIndex error: record not found, index: <parent>`.
- **Cause**: Shadow DB imported bundles starting at batch N, but batch N-1 was not imported.
- **Rule**: Always insert the parent batch skeleton before starting the relayer. Only `state_root` must be accurate.

### Trap 7: Relayer Nonce Desync
- **Symptom**: Tx sent but never mined; `eth_getTransactionReceipt` returns null forever.
- **Cause**: `pending_transaction` table retains nonces from previous runs that were never confirmed. Relayer initializes nonce from `maxDbNonce + 1`, which is ahead of the on-chain nonce.
- **Rule**: After any relayer crash or Anvil restart:
  ```sql
  DELETE FROM pending_transaction WHERE sender_address = '<finalize_sender>';
  ```
  Then restart the relayer.

### Trap 8: Production Proof Overwrite
- **Symptom**: `VerificationFailed` after importing production data.
- **Cause**: Import script copied `proof` columns from production RDS, overwriting locally-valid shadow proofs.
- **Rule**: **Never import `proof` columns**. Import only metadata, then reset `proving_status = 1` and re-prove locally.

### Trap 9: Anvil `eth_estimateGas` Rejects Fee Caps
- **Symptom**: `failed to get fee data, err: Out of gas: gas required exceeds allowance: 0`.
- **Cause**: Anvil's `eth_estimateGas` fails when `CallMsg` has `GasFeeCap`/`GasTipCap` set but `Gas` is 0 (Go Ethereum client's default).
- **Rule**: If testing relayer against Anvil and gas estimation fails, patch `estimategas.go` to strip fee caps from the `EstimateGas` call (see `LESSONS_LEARNED.md` for exact patch).

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
- [ ] **Start relayer with `--config <path>` AND `--min-codec-version 10`**
- [ ] Monitor logs for `finalizeBundle in layer1` success
- [ ] Verify `lastFinalizedBatchIndex` advanced on Anvil

## When Things Go Wrong

| Error / Symptom | Most Likely Cause | See |
|-----------------|-------------------|-----|
| `VerificationFailed(0x439cc0cd)` | Wrong verifier type or digest mismatch | Trap 1 |
| `ErrorIncorrectBatchHash(0x2a1c1442)` | Sparse `committedBatches`, end batch hash is zero | Trap 4 |
| `ErrorFinalizedIndexTooLarge(0x16465978)` | `nextUnfinalizedQueueIndex` too low or too high | Trap 5 |
| `record not found` (parent batch) | Parent batch not imported | Trap 6 |
| `Out of gas: gas required exceeds allowance: 0` | Anvil gas estimation bug with fee caps | Trap 9 |
| `Insufficient funds for gas * price + value` | Sender balance is 0 on Anvil | Trap 10 |
| Tx sent but never mined | Nonce desync (`pending_transaction` stale) | Trap 7 |
| Relayer exits with `Required flag "min-codec-version" not set` | Missing CLI flags | Trap 11 |
| Coordinator assigns but prover gets nothing | L2 RPC missing `debug_executionWitness` | README.md |
| `CoordinatorEmptyProofData` | Prover crashed; reset stuck tasks | README.md |

## Documentation Priority

When debugging, read docs in this order:

1. `docs/LESSONS_LEARNED.md` — fastest path to known solutions
2. `docs/GUIDE.md` — detailed setup and troubleshooting
3. `README.md` — quick reference for common commands
4. `../../AGENTS.md` (repo root) — cross-network rules and secrets reference
