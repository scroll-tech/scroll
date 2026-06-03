# Agent Instructions for Shadow Testing

> **Read this file first** before starting any shadow fork or shadow coordinator test.
> This directory contains hard-won knowledge from multiple debugging sessions. Blind experimentation will repeat documented mistakes.

## Pre-Flight Ritual (Mandatory)

Before executing a single command:

1. [ ] **Read `AGENTS.md`** (this file) — refresh the trap list.
2. [ ] **Read `docs/LESSONS_LEARNED.md`** — check if your planned task matches any documented failure mode.
3. [ ] **Read `docs/README.md`** — verify the specific section matching your task (e.g., "Real Verifier Deployment", "Multi-Bundle Relayer Finalize Test").
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
| Verifier | Copy from mainnet (`anvil_setCode`) | Must deploy fresh (`PostFeynman`) | `cast call <MVRV> "getVerifier(uint256,uint256)" 10 <batchIndex>` |

## Critical Traps (Do Not Skip)

### Trap 1: Wrong Verifier Contract
- **Symptom**: `VerificationFailed(0x439cc0cd)` even with correct digests.
- **Cause**: Deployed `ZkEvmVerifierPostEuclid` instead of `ZkEvmVerifierPostFeynman`.
- **Rule**: For guest v0.8.0+ proofs, **always use `PostFeynman`**.
- **Verification**: Extract digests from proof instances:
  ```python
  instances = base64.b64decode(proof_json['proof']['instances'])
  digest1 = '0x' + instances[384:416].hex()   # offset 384-416
  digest2 = '0x' + instances[416:448].hex()   # offset 416-448
  ```
  Then deploy with `protocolVersion = 10`.

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
- **Symptom**: `ErrorFinalizedIndexTooLarge(0x16465978)`.
- **Cause**: `nextUnfinalizedQueueIndex < totalL1MessagesPoppedOverall` because `nextCrossDomainMessageIndex` was not synced.
- **Rule**: On Sepolia, `nextCrossDomainMessageIndex` and `nextUnfinalizedQueueIndex` must be **equal** before finalizing.

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
- [ ] Fund owner and sender accounts
- [ ] Add prover EOA to `ScrollChain`
- [ ] Set `lastFinalizedBatchIndex` to `(first_target_batch - 1)`
- [ ] Set `lastCommittedBatchIndex` to mainnet value (do NOT reset to lastFinalized)
- [ ] (Sepolia) Sync `L1MessageQueueV2.nextCrossDomainMessageIndex == nextUnfinalizedQueueIndex`

### Phase 3: Verifier Deployment
- [ ] Extract digests from proof instances
- [ ] Deploy plonk verifier from `coordinator/build/bin/assets_v2/verifier.bin`
- [ ] Deploy `ZkEvmVerifierPostFeynman` with digests + `protocolVersion = 10`
- [ ] Register on `MultipleVersionRollupVerifier` via `updateVerifier(10, startBatch, verifier)`
- [ ] Verify with `getVerifier(10, batchIndex)`

### Phase 4: Coordinator + Prover
- [ ] Coordinator config points to correct `assets_v2/` directory
- [ ] Coordinator L2 RPC is internal/debug-enabled
- [ ] Prover config `base_url` uses correct S3 path (no `/releases/` for v0.8.0)
- [ ] Start coordinator, wait for `Start coordinator api successfully`
- [ ] Start prover(s), verify `Got task from coordinator`

### Phase 5: Relayer Finalize
- [ ] Build relayer with latest code
- [ ] Relayer config has `dry_run: false`, correct contract addresses
- [ ] Clear stale `pending_transaction` entries
- [ ] Reset target bundles to `rollup_status = 1`
- [ ] Start relayer
- [ ] Monitor logs for `finalizeBundle in layer1` success
- [ ] Verify `lastFinalizedBatchIndex` advanced on Anvil

## When Things Go Wrong

1. **`VerificationFailed(0x439cc0cd)`** → See Trap 1 (verifier contract type/digests).
2. **`ErrorIncorrectBatchHash(0x2a1c1442)`** → See Trap 4 (sparse `committedBatches`).
3. **`ErrorFinalizedIndexTooLarge(0x16465978)`** → See Trap 5 (L1MessageQueueV2 indices).
4. **`record not found` (parent batch)** → See Trap 6.
5. **Tx sent but never mined** → See Trap 7 (nonce desync).
6. **Coordinator says tasks assigned but prover gets nothing** → L2 RPC missing `debug_executionWitness`, or chunks have `codec_version = 5`.
7. **`CoordinatorEmptyProofData`** → Prover crashed; reset stuck tasks.

## Documentation Priority

When debugging, read docs in this order:

1. `docs/LESSONS_LEARNED.md` — fastest path to known solutions
2. `docs/README.md` — detailed setup and troubleshooting
3. `docs/QUICKSTART.md` — quick reference for common commands
4. `../../AGENTS.md` (repo root) — cross-network rules and secrets reference
