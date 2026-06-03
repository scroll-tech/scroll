# Shadow Testing Toolkit

One-command toolkit for running Scroll shadow fork tests against Anvil.

## Quick Start

### Docker (Recommended)

```bash
cd tests/shadow-testing

# Full pipeline
make docker-all CONFIG=mainnet BUNDLE_RANGE=17302:17305

# Or step by step:
make docker-env      CONFIG=mainnet BUNDLE_RANGE=17302:17305  # Postgres + import + Anvil
make docker-prove    CONFIG=mainnet BUNDLE_RANGE=17302:17305  # Coordinator + Prover
make docker-finalize CONFIG=mainnet BUNDLE_RANGE=17302:17305  # Relayer

# Stop all docker services
make docker-stop
```

### Bare-Metal

```bash
cd tests/shadow-testing

# Mainnet full pipeline
make all CONFIG=mainnet BUNDLE_RANGE=17297:17301

# Sepolia full pipeline
make sepolia-all CONFIG=sepolia BUNDLE_RANGE=13410:13414

# Or step by step:
make env      CONFIG=mainnet BUNDLE_RANGE=17297:17301   # Anvil + DB reset
make prove    CONFIG=mainnet BUNDLE_RANGE=17297:17301   # Start provers + wait
make finalize CONFIG=mainnet BUNDLE_RANGE=17297:17301   # Relayer + wait

# Check status
make status   CONFIG=mainnet BUNDLE_RANGE=17297:17301
make verify   CONFIG=mainnet BUNDLE_RANGE=17297:17301

# Clean up
make stop
make clean
```

## Directory Structure

```
.
├── Makefile                    # Main orchestration (bare-metal + docker)
├── docker-compose.yml          # Postgres + Coordinator + Relayer + Prover (Docker)
├── configs/
│   ├── mainnet.json            # Mainnet fork parameters
│   ├── sepolia.json            # Sepolia fork parameters
│   ├── coordinator.json        # Coordinator API config
│   └── relayer.json.template   # Relayer config template
├── scripts/
│   ├── 00-import-bundle-range.sh  # Import bundle range from production RDS
│   ├── 01-setup-anvil.sh          # Start Anvil fork + reset state + register verifier
│   ├── 02-prepare-db.sh           # Reset bundle/batch rollup_status in shadow DB
│   ├── 03-prover-up.sh            # Launch GPU prover(s)
│   ├── 04-wait-for-proofs.sh      # Poll DB until proving_status=4
│   ├── 05-run-relayer.sh          # Build + launch rollup relayer
│   ├── 06-wait-for-finalize.sh    # Poll Anvil until lastFinalizedBatchIndex reaches target
│   ├── 07-docker-orchestrate.sh   # One-command Docker orchestrator
│   └── lib/
│       └── anvil-utils.sh         # Shared helpers (colors, cast wrappers)
└── states/                        # Saved Anvil state files (gitignored)
```

## Prerequisites

- `cast`, `anvil` (Foundry)
- `jq`
- `psql` (PostgreSQL client)
- `go` (for building relayer)
- `cargo` + CUDA (for building prover, bare-metal only)
- Docker + Docker Compose (for dockerized pipeline)

## Configurations

> **⚠️ Secrets are not committed.** All config files in `configs/` are stored as `.template` files.
> You must copy them and fill in your own secrets before running any commands.

### Initial Setup

```bash
cd tests/shadow-testing

# Copy templates to actual config files
cp configs/mainnet.json.template configs/mainnet.json
cp configs/sepolia.json.template configs/sepolia.json
cp configs/coordinator.json.template configs/coordinator.json

# Edit the files and replace placeholders:
#   - YOUR_ALCHEMY_API_KEY   (in mainnet.json / sepolia.json fork.url)
#   - YOUR_SHADOW_DB_PASSWORD (in mainnet.json / sepolia.json db.dsn)
#   - YOUR_COORDINATOR_AUTH_SECRET (in coordinator.json auth.secret)
```

The `.gitignore` already ignores `configs/*.json`, so your actual configs with secrets will never be committed.

### Mainnet (configs/mainnet.json)

Pre-filled with addresses and parameters from the successful 5-bundle test:
- Fork URL: Alchemy mainnet
- Fork block: 25202217
- ScrollChain: `0xa13BAF...`
- Verifier: `0xb1F2C5...` (deployed with shadow proof digests)

### Sepolia (configs/sepolia.json)

Pre-filled with Sepolia contract addresses. Key differences from mainnet:
- **Fork URL**: Sepolia RPC instead of mainnet
- **Contract addresses**: Sepolia Scroll deployment
- **DB DSN**: Points to Sepolia shadow DB via port forward (`localhost:25432`)

Sepolia test follows the **exact same pattern** as mainnet:
```bash
make env CONFIG=sepolia BUNDLE_RANGE=13410:13414      # Setup Anvil fork Sepolia
make prove CONFIG=sepolia BUNDLE_RANGE=13410:13414    # Re-prove with v0.8.0 guest
make finalize CONFIG=sepolia BUNDLE_RANGE=13410:13414 # Finalize on Anvil
```

**Note**: Sepolia shadow DB (`scroll-sepolia.c9juduqkmttk.us-west-2.rds.amazonaws.com`) is available via port forwarding at `localhost:25432`.

**Prover version**: Sepolia DB contains old proofs (v0.7.3 guest). `02-prepare-db.sh` **resets `proving_status` to 1** so the coordinator will re-assign tasks to the prover, which will generate fresh v0.8.0 proofs.

**Verifier**: Sepolia L1 does not have a registered codec v10 verifier (`latestVerifier[10] = 0x0`). The script attempts to copy the shadow-compatible verifier from the mainnet Anvil instance. If unavailable, you must deploy a verifier matching your locally-generated proofs.

**⚠️ L2 RPC Requirement**: Coordinator chunk task generation requires `debug_executionWitness` on the L2 RPC. The public Sepolia RPC (`https://sepolia-rpc.scroll.io`) **does not expose this method** (returns `-32053: API key is not allowed to access method`). You must use an internal/debug-enabled Scroll Sepolia L2 node. Update the coordinator config's `l2.l2geth.endpoint` before starting the coordinator:
```json
"l2geth": { "endpoint": "https://l2geth-rpc-proxy.sepolia.aws.scroll.io" }
```

## How It Works

### Phase 1: Environment Setup (`make env`)

1. **Anvil fork** — Forks Ethereum mainnet at the specified block
2. **State reset** — Uses `anvil_setStorageAt` to:
   - Reset `lastFinalizedBatchIndex` to target - 1
   - Reset `nextUnfinalizedQueueIndex` to 0
3. **Verifier registration** — Impersonates owner, calls `updateVerifier` on `MultipleVersionRollupVerifier`
4. **Prover authorization** — Adds prover EOA via `addProver`
5. **DB reset** — Sets `rollup_status = 1` (`RollupPending`) for target bundles/batches

### Phase 2: Proving (`make prove`)

1. **Prover launch** — Starts `prover` binary per GPU with `CUDA_VISIBLE_DEVICES`
2. **Wait loop** — Polls shadow DB every 30s until all bundles have `proving_status = 4`

### Phase 3: Finalization (`make finalize`)

1. **Relayer launch** — Builds `rollup_relayer`, renders config from template, starts process
2. **Wait loop** — Polls Anvil every 15s until `lastFinalizedBatchIndex` >= target batch

## Key Design Decisions

### Why not everything in Docker Compose?

- **Anvil** — Needs fork URL/block parameters that vary per test. Script is more flexible.
- **Prover** — Requires GPU (`nvidia-smi` at compile time, CUDA at runtime). Kept bare-metal for Phase 1; Dockerized in Phase 2.
- **Relayer** — Needs freshly built binary from source. Script handles `go build` automatically.

### Why `--state-file`?

After the first manual setup (impersonating owner, registering verifier, etc.), the Anvil state can be saved and reused. Subsequent runs skip all the setup steps:

```bash
# First time: full setup + save
./scripts/01-setup-anvil.sh --state-file states/mainnet.state.json

# Subsequent runs: just load
anvil --state states/mainnet.state.json --port 18545
```

### Why reset `rollup_status`?

The shadow DB is imported from mainnet production data. Bundles already have `rollup_status = 5` (`RollupFinalized`). The relayer's `GetFirstPendingBundle` only queries `rollup_status = 1` (`RollupPending`). Resetting is mandatory before relayer will pick them up.

## Troubleshooting

### `ErrorCallerIsNotSequencer`

The relayer's batch committer sends `commitBatch` from the commit EOA. If this EOA is not a sequencer, the transaction reverts. This is **expected** in shadow testing — the batch committer fails but the bundle finalizer continues independently. To fix, add the commit EOA as a sequencer:

```bash
# In 01-setup-anvil.sh, ensure --commit-eoa is set
```

### `lastFinalizedBatchIndex` not advancing

Check relayer logs:
```bash
tail -f tests/shadow-testing/.work/relayer-mainnet.log
```

Common causes:
- `rollup_status` not reset to 1
- Prover not authorized (`isProver = false`)
- Verifier not registered (`latestVerifier[10] = 0x0`)
- `nextUnfinalizedQueueIndex` mismatch

### Stale Anvil state from Docker (`L1MessageQueueV2` implementation corrupted)

**Symptom**: `cast call L1MessageQueueV2 "nextUnfinalizedQueueIndex()(uint256)"` fails with `TransparentUpgradeableProxy: admin cannot fallback to proxy target`, or returns strange values. `eth_getStorageAt` shows implementation slot = `0x1111111111111111111111111111111111111112`.

**Root cause**: The Docker `shadow-anvil` container persists state to `./states/anvil.state.json`. If you (or a previous run) ever modified contract storage via `anvil_setStorageAt` or `anvil_setCode`, that mutation is saved and reloaded on the next container start — even after `docker compose down`.

**Fix**:
```bash
# Stop the stale container
docker stop shadow-anvil && docker rm shadow-anvil

# Delete the corrupted state
rm -f tests/shadow-testing/states/anvil.state.json

# Also clear Foundry's RPC cache (may contain stale storage reads)
rm -rf ~/.foundry/cache/rpc/mainnet/*

# Restart fresh
cd tests/shadow-testing && make env CONFIG=mainnet BUNDLE_RANGE=17302:17305
```

### Manually seeding `committedBatches` mapping slots

**Symptom**: You try to manually set `committedBatches[517766]` via `anvil_setStorageAt`, but `eth_call` / transaction execution still sees `bytes32(0)`.

**Root cause**: `anvil_setStorageAt` on **mapping slots** is visible to `eth_getStorageAt` but is **cached and ignored** by `eth_call` / `eth_sendTransaction` during contract execution. This is a known Anvil limitation/bug.

**The real fix**: Don't manually set `committedBatches` at all.
- `committedBatches` is **sparse** starting from EuclidV2: only the *last* batch of each commit tx is stored.
- If you fork at a block where the batches were already committed on mainnet, the correct entries already exist on-chain.
- Only `committedBatches[lastCommittedBatchIndex]` (the parent hash) is needed for `finalizeBundlePostEuclidV2`, and that is already present if `lastCommittedBatchIndex` is left at its mainnet value.

**Correct approach**:
1. Keep `lastCommittedBatchIndex` at its **mainnet value** (do NOT reset it to `lastFinalizedBatchIndex`).
2. Do NOT attempt to manually seed any `committedBatches[...]` mapping entries.
3. Only reset `lastFinalizedBatchIndex` to one less than your first target batch.

This is now supported via `configs/mainnet.json`:
```json
{
  "reset": {
    "last_finalized_batch_index": 517765,
    "last_committed_batch_index": 517819,
    ...
  }
}
```

### `CoordinatorGetTaskFailure: API key is not allowed to access method`

The coordinator's `gen_universal_task` needs L2 RPC method `debug_executionWitness` (via `dump_block_witness`) to build block witnesses for chunk proving. Public Sepolia RPCs reject this debug API with error code `-32053`. 

**Fix**: Update the coordinator config with an internal/debug-enabled L2 endpoint:
```bash
# In coordinator config (e.g., /tmp/coordinator-sepolia.json)
"l2geth": { "endpoint": "https://l2geth-rpc-proxy.sepolia.aws.scroll.io" }
```
Then restart the coordinator.

### `failed to get parent batch for batch task id: record not found`

Bundle proving requires the **parent batch** of the bundle's first batch to exist in the DB. If you import a limited batch range (e.g., 127917:127927), the parent of the first batch (127916) is missing.

**Root cause**: Coordinator's `bundle_prover_task.go` calls `GetBatchByHash(parentBatchHash)` to retrieve `prevStateRoot` for bundle public inputs. If the parent batch is absent, task generation fails immediately with `record not found`.

**Why this happens in shadow testing**: Production DB contains millions of batches. Shadow testing only imports a small slice (e.g., bundles 13410–13414 → batches 127917–127927). The first batch in this slice (127917) still references its real parent (127916), which lies outside the imported range.

**Correct fix**: Insert the missing parent batch with its **real state_root** derived from on-chain data:

```sql
-- Step 1: Derive the real state_root
-- batch N-1's post-state_root = batch N's first chunk's parent_chunk_state_root
SELECT parent_chunk_state_root FROM chunk WHERE index = 857862;
-- Returns: 0x20d0b5f742be811d3953caee978b2d106c423c21b3a406bfd019ae3cc4c812c5

-- Step 2: Insert parent batch skeleton (only state_root must be accurate)
INSERT INTO batch (
    index, hash, start_chunk_index, start_chunk_hash, end_chunk_index, end_chunk_hash,
    state_root, withdraw_root, parent_batch_hash, batch_header,
    chunk_proofs_status, proving_status, rollup_status, oracle_status,
    total_l1_commit_gas, total_l1_commit_calldata_size, total_attempts, active_attempts,
    data_hash, codec_version, enable_compress
) VALUES (
    127916,
    '0x5c6de7d03d55f9a1fcd1cc6a492758eaa62b313b2479594d51ef0b17822a5e89',
    857861, '0x00...', 857861, '0x00...',
    '0x20d0b5f742be811d3953caee978b2d106c423c21b3a406bfd019ae3cc4c812c5',
    '0x00...', '0x00...', '\x',
    1, 4, 1, 1, 0, 0, 0, 0, '0x00...', 10, false
);
```

**Note**: Only `state_root` must be accurate because the coordinator code path only reads `parentBatch.StateRoot`. Other fields (`withdraw_root`, `batch_header`, etc.) are not accessed during bundle task generation and can remain as placeholders. If you have access to the full shadow DB, importing the complete parent batch record is the most rigorous approach.

### Prover crash: `thread overflowed its stack`

OpenVM proving can exhaust the default Rust thread stack (8MB). The prover may crash mid-proof with `fatal runtime error: stack overflow`.

**Fix**: Launch prover with increased stack size:
```bash
RUST_MIN_STACK=33554432 ./prover --config prover.json
```

### `CoordinatorEmptyProofData` after task assignment

If a prover process dies while proving, its assigned task remains in `proving_status=2` with `active_attempts>0`. New prover instances cannot pick up these tasks.

**Fix**: Reset stuck tasks in DB:
```sql
UPDATE chunk SET proving_status=1, active_attempts=0, total_attempts=0 WHERE index IN (...);
DELETE FROM prover_task WHERE task_id IN (...);
```

### Port conflicts

```bash
# Check what's using port 18545
lsof -ti :18545
# Or use a different port
make env CONFIG=mainnet ANVIL_RPC=http://localhost:18546
```

## Mainnet Shadow Fork — Final Manual Finalization (Bundles 17302–17305)

This section documents what happened when we attempted to finalize **bundles 17302–17305** (batches 517766–517771) on the Mainnet shadow fork. It serves as a companion to the Sepolia section and highlights traps that are unique to (or much more severe on) Mainnet.

### Plan vs Reality

| Step | Plan (`make all`) | What Actually Happened |
|------|-------------------|------------------------|
| **Env setup** | `make env` → one-command Anvil + DB reset | ✅ Ran `01-setup-anvil.sh` manually. Worked after adding `--last-committed` support. |
| **Proving** | `make prove` → bare-metal prover waits for `proving_status=4` | ✅ Docker coordinator + GPU prover (`shadow-prover-gpu-0`) generated all proofs correctly. |
| **Finalization** | `make finalize` → relayer sends finalize txs automatically | ❌ **Relayer never successfully sent finalize txs.** We ended up manually sending all 4 transactions via `cast send` + `anvil_impersonateAccount`. |

### Why the Relayer Failed

The bare-metal relayer (`05-run-relayer.sh`) had multiple integration issues:

1. **Hardcoded dev keys** — The relayer template uses `COMMIT_KEY` and `FINALIZE_KEY` placeholders that were never properly injected in our environment.
2. **Config rendering gaps** — The rendered relayer config was missing or had incorrect fields (e.g., `validium_mode`, L1 RPC endpoints).
3. **Commit noise drowning finalize errors** — Even when the relayer started, its `batch_submitter` continuously retried `commitBatch` for already-committed batches, generating so much log noise that finalize errors were buried.
4. **No `--bundle-range` support** — `05-run-relayer.sh` does not accept `--bundle-range`. It tries to process *all* pending bundles in the DB, which on Mainnet meant attempting to commit/finalize hundreds of unrelated bundles.

**Verdict**: For targeted shadow fork tests (a specific 4-bundle window), the relayer is overkill and under-configured. Manual `eth_sendTransaction` is faster and more debuggable.

### How the Manual Finalization Worked

We wrote a Python script that:

1. **Queried the shadow DB** for each bundle's `proof`, `batch_header`, and `end_batch_index`.
2. **Extracted proof metadata** from the JSON proof blob (`proof` is a serialized `OpenVMBundleProof`):
   - `post_state_root`, `withdraw_root`, `msg_queue_hash` from `metadata.bundle_info`
   - `instances` and `proof` from `proof.proof` (base64-decoded)
3. **Reconstructed `aggrProof`** as `instances[:384] + proof_bytes` (exactly what `OpenVMBundleProof.Proof()` returns in Go).
4. **Derived `totalL1MessagesPoppedOverall`** by brute-forcing `L1MessageQueueV2.getMessageRollingHash(index)` until the hash matched `msg_queue_hash`.
   - Bundle 17302: `998278`
   - Bundles 17303–17305: `998279`
5. **ABI-encoded `finalizeBundlePostEuclidV2`** calldata and sent via `eth_sendTransaction` with `anvil_impersonateAccount`.

All 4 transactions succeeded:
- Bundle 17302: `0x53f3a0…` (gas: 156,403)
- Bundle 17303: `0x2aa585…` (gas: 123,247)
- Bundle 17304: `0x4d9774…` (gas: 117,120)
- Bundle 17305: `0x7a6373…` (gas: 117,168)

### Trap #1: Wrong Function Selector

The single biggest time sink was using the **wrong selector** for `finalizeBundlePostEuclidV2`.

- **Wrong**: `0x7a3f8e9a` (where this came from is still unclear — likely a copy-paste error from an earlier session)
- **Correct**: `0xc1aa4e19`

**Symptom**: `eth_call` returns `execution reverted, data: "0x"` (empty revert data). The proxy's DELEGATECALL to the implementation uses only ~235 gas and returns immediately.

**Lesson**: When you see an empty revert with almost no gas consumed, **suspect the selector first**. Always verify with:
```bash
cast sig "finalizeBundlePostEuclidV2(bytes,uint256,bytes32,bytes32,bytes)"
```

### Trap #2: Verifier Mismatch — Digests *and* Plonk Verifier

This is the most subtle trap. There are **two** independent verifier components:

1. **`ZkEvmVerifierPostEuclid` / `ZkEvmVerifierPostFeynman`** — the Solidity wrapper that assembles public inputs and calls the plonk verifier.
2. **The plonk verifier contract** — a ~18KB (new guest) or ~37KB (mainnet legacy) EVM bytecode that performs the actual SNARK verification.

**Mainnet's registered verifier** (`0x0dE180…`) has:
- Digests: `0x009160…` / `0x009305…`
- Plonk verifier: `0x749fC77A…` (37KB legacy bytecode)

**Our locally-generated proofs** (guest v0.8.0) require:
- Digests: extracted from the proof's `instances` array (see below)
- Plonk verifier: `coordinator/build/bin/assets_v2/verifier.bin` (18KB new bytecode)

**Symptom**: `VerificationFailed(0x439cc0cd)` even when digests *appear* correct. This happens because the plonk verifier bytecode itself differs between mainnet (37KB) and the new guest release (18KB). The wrapper may pass the right digests, but the underlying plonk verifier rejects the proof format.

**How to determine the correct verifier for your proofs**

The proof's `instances` array tells you exactly which wrapper + plonk verifier you need:

```python
import base64, json

# proof_json from coordinator/bundle table
instances = base64.b64decode(proof_json['proof']['instances'])
print(f"instances length: {len(instances)} bytes")  # e.g. 1472

# For ZkEvmVerifierPostEuclid (new guest v0.8.0+):
#   1472 bytes = 12 accumulators (384) + 2 digests (64) + 32 publicInputHash bytes (1024)
#   = 46 × 32-byte Fr elements
digest1 = '0x' + instances[384:416].hex()
digest2 = '0x' + instances[416:448].hex()
```

If `len(instances) == 1472` (or more generally `12+2+32 = 46` words), your proof is for **`ZkEvmVerifierPostEuclid`**. The wrapper computes `keccak256(publicInput)` and feeds each of the 32 hash bytes as a separate field element to the plonk verifier.

If `len(instances)` were smaller (e.g. 12+13 = 25 words), the proof would be for `ZkEvmVerifierPostFeynman`, which decomposes public inputs into 13 field elements directly.

**Fix — Automated deployment**

A script handles all three steps (deploy plonk, deploy wrapper, register):

```bash
# From tests/shadow-testing/
./scripts/02-deploy-verifier.sh --config configs/mainnet.json
```

What it does:
1. Reads `coordinator/build/bin/assets_v2/verifier.bin` and deploys it as the new plonk verifier via `cast send --create`.
2. Queries the shadow DB for the bundle proof, base64-decodes `instances`, and extracts `digest1`/`digest2` from offsets `[384:416]` and `[416:448]`.
3. Compiles and deploys `ZkEvmVerifierPostFeynman` (not `PostEuclid`!) with `forge create` using the new plonk address + extracted digests + `protocolVersion = 10`.
4. Calls `MultipleVersionRollupVerifier.updateVerifier(10, startBatch, newWrapper)`.
5. Updates `configs/mainnet.json` → `contracts.deployed_verifier`.

**Why `ZkEvmVerifierPostFeynman` and not `PostEuclid`**

`PostFeynman` computes `keccak256(abi.encodePacked(protocolVersion, publicInput))`, where `protocolVersion = (domain << 6) + stf_version` = `(0 << 6) + 10 = 10` for Scroll domain + V10 STF version. This matches exactly how the prover computes `bundle_pi_hash` in `pi_hash_versioned()`.

`PostEuclid` computes `keccak256(publicInput)` *without* the `protocolVersion` prefix, so the hash never matches the proof's embedded `bundle_pi_hash`. The 32-byte `protocolVersion` padding is visible in the proof's `instances` array at offset `[448:1472]` (32 words, each containing one byte of the hash).

**Why you cannot reuse the mainnet plonk verifier**

Mainnet's plonk verifier (`0x749fC77A…`, 37KB) was compiled for the old Halo2 circuit. The new guest v0.8.0 uses a different Halo2 outer circuit with a smaller verification key. The 18KB `verifier.bin` from `assets_v2/` is the *only* plonk verifier that understands the new proof format. Copying mainnet bytecode via `anvil_setCode` only works if you are verifying **mainnet-generated proofs**; it will always fail for **locally-generated proofs** from a newer guest.

### Trap #3: `committedBatches` Is Sparse (EuclidV2)

We initially worried that `committedBatches[517768]` was `0x0` (bundle 17303's first batch). In pre-Euclid code, every batch hash is stored. In EuclidV2, `_commitBatchesFromV7` only stores the **last batch hash** of each commit transaction.

**Lesson**: Do NOT manually seed `committedBatches` mapping slots. Anvil's fork already has the correct sparse entries from mainnet. The contract only checks `committedBatches[batchIndex]` where `batchIndex` is the **last batch** of the bundle being finalized.

### Trap #4: `anvil_setStorageAt` on Mapping Slots Is Ignored

Anvil has a known bug: `anvil_setStorageAt` on mapping slots is visible to `eth_getStorageAt` but is **cached and ignored** by `eth_call` / `eth_sendTransaction`. We hit this early when trying to manually inject batch hashes.

**Lesson**: Don't fight the fork. If mainnet already committed the batches at the fork block, the storage is already correct.

### Trap #5: `totalL1MessagesPoppedOverall` Cannot Be Zero

The `publicInputs` encode `messageQueueHash = getMessageRollingHash(totalL1MessagesPoppedOverall - 1)`. Setting `totalL1MessagesPoppedOverall = 0` makes `messageQueueHash = 0x0`, which does not match the proof's expected public inputs.

**How to derive it**:
```python
# Brute-force search against L1MessageQueueV2
for i in range(start_guess, end_guess):
    hash = cast_call(MQ, "getMessageRollingHash(uint256)(bytes32)", str(i))
    if hash == target_msg_queue_hash:
        total_l1 = i + 1
        break
```

### Makefile Fixes Applied

During the session we fixed several Makefile bugs:

1. **`anvil-up` missing `--last-committed`**
   - `01-setup-anvil.sh` supports `--last-committed`, but the Makefile never passed it.
   - Fix: Add `LAST_COMMITTED` variable and pass `--last-committed "$(LAST_COMMITTED)"`.

2. **`relayer-up` had erroneous `--bundle-range`**
   - `05-run-relayer.sh` does **not** accept `--bundle-range`.
   - Fix: Remove the argument.

3. **`wait-finalize` missing `--bundle-range`**
   - `06-wait-for-finalize.sh` **does** need `--bundle-range` to know the target batch.
   - Fix: Add `--bundle-range "$(BUNDLE_RANGE)"`.

4. **`anvil-up` had erroneous `--bundle-range`**
   - `01-setup-anvil.sh` does not accept `--bundle-range`.
   - Fix: Remove it.

---

## Sepolia Shadow Fork — Detailed Lessons Learned

This section documents the **real-world debugging journey** of the Sepolia shadow fork (bundles 13410–13414, batches 127917–127927) and contrasts it with the smoother Mainnet test. It is meant to save future operators from repeating the same mistakes.

### Why Sepolia Was Harder Than Mainnet

| Dimension | Mainnet (bundles 17297–17301) | Sepolia (bundles 13410–13414) | Impact |
|-----------|------------------------------|-------------------------------|--------|
| **DB scope** | Shadow DB imported *only* the test range (6 blocks → 1 bundle). | Shadow DB is a **full production snapshot** (`sepolia_scroll`), containing live batches up to 128080+. | Relayer continuously tries to commit new live batches, generating massive log noise and distracting from the actual finalize errors. |
| **Batch numbering** | Batch ~517760 (small numbers). | Batch ~127917 (much larger). | Higher cognitive load; `committedBatches` slot math is the same but easier to mistype. |
| **`committedBatches` density** | Pre-EuclidV2 or well-covered by setup. | **EuclidV2 sparse commits**: `_commitBatchesFromV7` only stores the *last* batch hash of each commit tx. | If you forget to seed `committedBatches` for **every bundle end batch** (127918, 127921, 127923, 127925, 127927), finalize reverts with `ErrorIncorrectBatchHash` (`0x2a1c1442`). |
| **Verifier availability** | Mainnet Anvil had a pre-deployed shadow-compatible verifier (`0xb1F2C5...`) whose bytecode could be copied. | Sepolia L1 had **no codec v10 verifier** registered (`latestVerifier[10] = 0x0`). | Had to deploy `MockZkEvmVerifierV2` manually and register it via `updateVerifier`. |
| **L1MessageQueueV2 state** | Reset `nextUnfinalizedQueueIndex = 0` was sufficient. | Production values were `nextCrossDomainMessageIndex = 1091211` and `nextUnfinalizedQueueIndex = 1091211`. | If only `nextUnfinalizedQueueIndex` is reset without syncing `nextCrossDomainMessageIndex`, `finalizePoppedCrossDomainMessage` reverts with `ErrorFinalizedIndexTooLarge` (`0x16465978`). |
| **BlobSidecar version** | Not triggered (or V0 by default). | Anvil 1.0.0 **cannot decode BlobSidecar V1**; relayer commit tx serialization fails. | Required `fusaka_timestamp: 2000000000` (far future) in relayer config to force V0 blobs. |

### The Full Error Cascade (Timeline)

Understanding the *order* in which errors appeared is critical. Fixing them out of order wastes hours.

**Step 1 — Verifier missing**
- Relayer finalize fails: `execution reverted: \x16FYx` → `ErrorIncorrectBatchHash` (`0x2a1c1442`).
- But the *real* root cause was not batch hash. It was that `MultipleVersionRollupVerifier.latestVerifier[10]` returned `0x0`, causing `verifyBundleProof` to hit a fallback path that consumed all gas and returned a mangled revert reason.
- **Lesson**: Always check `latestVerifier[codec]` first. A missing verifier can surface as a cryptic batch-hash error because the trace runs out of gas inside the verifier call.

**Step 2 — `committedBatches` sparse slots**
- After deploying the mock verifier, relayer finalize still failed with `ErrorIncorrectBatchHash`.
- `_finalizeBundlePostEuclidV2` calls `_loadBatchHeader(batchIndex)`, which checks `committedBatches[batchIndex]`. In EuclidV2, only the *end* batch of each commit tx is stored.
- **Fix**: For each bundle end batch, compute the mapping slot (`cast index uint256 <batch> 157`) and write the expected batch header hash directly into Anvil storage.

**Step 3 — `lastFinalizedBatchIndex` state drift**
- Bundle 13410 was manually finalized with `cast send` (tx `0x99ba...`). This correctly moved `lastFinalizedBatchIndex` from 127915 → 127918.
- Later, re-running `01-setup-anvil.sh` (or any `anvil_setStorageAt` on miscData) **overwrote** `lastFinalizedBatchIndex` back to 127915, or accidentally forward to 127927.
- When relayer restarted, it saw DB `rollup_status = 3/5` for bundles already finalized on-chain, or saw `lastFinalizedBatchIndex = 127927` and threw `ErrorBatchIsAlreadyVerified` (`0x92d31550`) for every bundle.
- **Lesson**: Treat `lastFinalizedBatchIndex` as a **critical shared state**. Never run setup scripts after manual finalization without first snapshotting the current value. Use `cast call ScrollChain "lastFinalizedBatchIndex()(uint256)"` before every state mutation.

**Step 4 — Relayer commit noise**
- Because the Sepolia DB contains batches 128011+, the relayer's `batch_submitter` repeatedly tries to commit them to Anvil.
- Each attempt fails with `parent batch index is not equal to current batch index - 1` (Anvil has sparse commits) or `ErrorIncorrectBatchHash`.
- These errors are **expected and harmless** for a shadow fork, but they flood the log file (5.6 MB in a few hours) and make it hard to spot the finalize errors.
- **Lesson**: For Sepolia, consider temporarily raising `batch_submission.timeout` or disabling the batch committer if you only care about finalize. The `finalize_sender` operates independently.

**Step 5 — `Out of gas` during `estimateGas`**
- At one point, relayer reported `Out of gas: gas required exceeds allowance: 0` for bundle 13410 finalize.
- This happened because Anvil's `estimateGas` could not execute the transaction successfully (due to `committedBatches` or `L1MessageQueueV2` mismatch), so it returned gas=0 instead of a meaningful revert reason.
- **Lesson**: When `estimateGas` returns 0, do not assume a gas-limit problem. Use `cast run --rpc-url ... --trace` with the raw calldata to see the actual revert path.

### Correct State Setup Checklist (Sepolia)

Before starting the relayer, verify **all** of these on Anvil:

```bash
SCROLL_CHAIN="0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0"
ANVIL_RPC="http://localhost:18546"

# 1. Verifier registered for codec 10
cast call "$SCROLL_CHAIN" "verifier()(address)" --rpc-url "$ANVIL_RPC"
cast call <VERIFIER_ADDR> "latestVerifier(uint256)" 10 --rpc-url "$ANVIL_RPC"

# 2. Prover EOA authorized
cast call "$SCROLL_CHAIN" "isProver(address)(bool)" \
  "0x410E7FD80a3Fc1E62A4D3450d11b71b812006eB9" --rpc-url "$ANVIL_RPC"

# 3. lastFinalizedBatchIndex is exactly one less than first target batch
cast call "$SCROLL_CHAIN" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$ANVIL_RPC"
# Expected: 127915

# 4. committedBatches set for EVERY bundle end batch
for batch in 127918 127921 127923 127925 127927; do
  slot=$(cast index uint256 "$batch" 157)
  hash=$(cast rpc eth_getStorageAt "$SCROLL_CHAIN" "$slot" "latest" --rpc-url "$ANVIL_RPC")
  echo "batch $batch: $hash"
done
# Expected: non-zero 32-byte values

# 5. L1MessageQueueV2 indices synced
QUEUE="0xA0673eC0A48aa924f067F1274EcD281A10c5f19F"
cast call "$QUEUE" "nextCrossDomainMessageIndex()(uint256)" --rpc-url "$ANVIL_RPC"
cast call "$QUEUE" "nextUnfinalizedQueueIndex()(uint256)" --rpc-url "$ANVIL_RPC"
# Expected: both equal (e.g., 1091211)
```

## When to Use Manual Transaction Scripts vs Relayer

During the Sepolia debug, we wrote a Python script (`/tmp/manual_finalize.py`) to send raw `finalizeBundlePostEuclidV2` transactions directly. Here is why, and what we learned.

### Why we reached for Python

1. **Relayer error messages are opaque**
   - Relayer wraps `estimateGas` failures as: `failed to get fee data, err: execution reverted: custom error 0x92d31550`.
   - You cannot tell from the log whether it is `ErrorBatchIsAlreadyVerified`, `ErrorIncorrectBatchHash`, or `ErrorCallerIsNotProver` without manually decoding the selector.

2. **Relayer retries mask state changes**
   - Relayer retries every 15 seconds. If you fix the underlying Anvil state while the relayer is in a retry loop, it may take minutes before the next attempt. A Python script gives instant feedback.

3. **Gas estimation can lie**
   - When Anvil state is inconsistent (e.g., missing `committedBatches`), `estimateGas` sometimes returns `0` or throws `Out of gas`. A manual script with a hardcoded gas limit (500k) bypasses this entirely and lets you use `cast run` to see the *real* revert trace.

4. **Impersonation flexibility**
   - The Python script used `anvil_impersonateAccount` to send from the finalize EOA without needing its private key in the script. This is faster than reconfiguring the relayer's signer.

### What the Python script actually did

- Queried the shadow DB for `bundle.proof`, `batch.batch_header`, `batch.state_root`, `batch.withdraw_root`, and chunk L1 message counts.
- Reconstructed the `aggr_proof` bytes by concatenating base64-decoded `instances` (first 384 bytes) with `proof`.
- ABI-encoded the `finalizeBundlePostEuclidV2` calldata manually.
- Sent 4 transactions (bundles 13411–13414) via `eth_sendTransaction` with fixed gas.

**Result**: The transactions were mined but reverted (status=0). This proved that the calldata construction was correct, and the problem was **Anvil state** — not the relayer, not the proof format, and not the contract logic.

### The verdict: Relayer vs Manual

| Situation | Use Relayer | Use Manual Script |
|-----------|-------------|-------------------|
| Clean Anvil state, standard flow | ✅ | ❌ |
| Debugging cryptic reverts | ❌ (too slow/opaque) | ✅ (instant, traceable) |
| Verifying calldata correctness | ❌ | ✅ |
| Batch finalize after state fix | ✅ (handles nonces, access lists, DB updates) | ❌ |

**Bottom line**: Manual scripts are a **diagnostic scalpel**; the relayer is the **production hammer**. We used Python to bisect the problem space rapidly, then let the relayer take over once the state was clean. In the final successful run, the relayer itself sent all 4 remaining finalize transactions (txs `0xd19f...`, `0x9654...`, `0xed0a...`, `0x61a3...`) without issue.

## Docker Pipeline Details

The Docker pipeline (`07-docker-orchestrate.sh`) provides a fully containerized alternative to the bare-metal approach:

### Phase: `docker-env`
1. Start `shadow-postgres` container on `:5433`
2. Import bundle range from production RDS via `00-import-bundle-range.sh`
3. Run bare-metal Anvil fork (`01-setup-anvil.sh`)
4. Reset ScrollChain state and register verifier

### Phase: `docker-prove`
1. Start `shadow-coordinator` container (downloads verifier assets on first run)
2. Render prover config from `configs/mainnet.json`
3. Start `shadow-prover-gpu-0` container with GPU reservation
4. Poll DB until all bundles have `proving_status = 4`

### Phase: `docker-finalize`
1. Render relayer config from template
2. Start `shadow-relayer` container using pre-built `rollup_relayer` binary
3. Poll Anvil until `lastFinalizedBatchIndex` reaches target

### Known Docker Fixes

| Issue | Fix |
|-------|-----|
| Coordinator assets path mismatch | Mount `assets_v2` → `/app/assets` (not `/app/assets_v2`) |
| Prover config path | Mount to `/prover/conf/config.json` (prover binary defaults to `conf/config.json`) |
| Prover work directory | Prover image `WorkDir` is `/prover`, not `/app` |
| Prover image tag | Use `e2e-test` tag (no `latest` exists locally) |
| Docker profiles | Use `--profile coordinator --profile prover` when starting prover |
| Coordinator L2 RPC | ✅ **Primary**: `https://l2geth-rpc-proxy.mainnet.aws.scroll.io/` (internal, supports `debug_executionWitness`)<br>⚠️ **Fallback**: `https://mainnet-rpc.scroll.io` (public, may not support `debug_executionWitness`) |
| Prover health check | Use `docker exec` instead of `localhost:10080` (no port mapping needed) |

## Roadmap

- [x] Phase 1: Script everything
- [x] Phase 2: Dockerize prover (multi-GPU compose)
- [x] Phase 3: Sepolia shadow fork support
- [x] Phase 4: One-command Docker orchestrator
