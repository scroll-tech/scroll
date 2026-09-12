# Snapshot Replay Mode Guide — Shadow Coordinator + Prover Testing

Snapshot replay forks a **historical** ETH mainnet block, imports a fixed bundle range, and proves + finalizes ~N bundles. Use it when follow mode is the wrong tool: reproducing a specific incident, debugging one specific bundle, Sepolia testing, or targeted codec-migration checks. For the primary acceptance workflow (following live mainnet), see the [Follow Mode Guide](../follow/GUIDE.md).

- Shared scripts: [`../lib/`](../lib/) · runtime state: `../.work/` (shared with follow mode)
- Pitfalls: [`./TROUBLESHOOTING.md`](./TROUBLESHOOTING.md) (snapshot) and [`../docs/COMMON-TROUBLESHOOTING.md`](../docs/COMMON-TROUBLESHOOTING.md) (mode-independent) — **read both before starting**
- Version-sensitive facts (S3 prefixes, digest encodings, codec block thresholds): [`../docs/CURRENT-STACK.md`](../docs/CURRENT-STACK.md)
- Historical dry-run / relayer experiment results: [`../../../docs/testing_reports/snapshot-early-dryrun-and-relayer-tests.md`](../../../docs/testing_reports/snapshot-early-dryrun-and-relayer-tests.md)

## Architecture

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  Production RDS  │     │  Shadow DB       │     │  Shadow          │
│  (read-only via  │────▶│  (local :5433)   │────▶│  Coordinator     │
│   port-forward)  │     │                  │     │  (localhost:8390)│
└──────────────────┘     └──────────────────┘     └────────┬─────────┘
                                                            │ assigns tasks
                                                            ▼
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  L2 RPC          │     │  Local Prover    │     │  Verifier Assets │
│  (mainnet-rpc.   │◀────│  (GPU/CPU)       │     │  (/tmp/shadow-   │
│   scroll.io)     │     │                  │     │   verifier-assets)│
└──────────────────┘     └──────────────────┘     └──────────────────┘
```

## Prerequisites

- GPU with CUDA support (tested on RTX 3090/4090), ~50 GB disk, 16 GB+ RAM
- Docker + docker-compose, `psql`, Rust toolchain (prover), foundry (`cast`/`forge`/`anvil`)
- Access to the production RDS (read-only, via tunnel — DSN in `local-secrets.md`)
- L2 RPC with `debug_executionWitness` (internal proxy; public RPCs block debug methods — COMMON Trap 3)

## Quick Start

The Makefile drives the whole pipeline; `snapshot/README.md` has the full target list.

```bash
cd tests/shadow-testing/snapshot
make docker-all CONFIG=mainnet BUNDLE_RANGE=17302:17305   # Docker (recommended)
make all        CONFIG=mainnet BUNDLE_RANGE=17297:17301   # bare-metal: env → prove → finalize
make sepolia-all CONFIG=sepolia BUNDLE_RANGE=<start:end>  # Sepolia pipeline
make help
```

Copy configs from templates first (`configs/mainnet.json.template` etc.) and fill in secrets. The step-by-step below is the manual equivalent, useful when you need fine control.

## Step-by-Step Setup

### Step 1: IDC port-forward to production RDS

Mainnet RDS should be reachable on `localhost:15432` (or use the direct address from `local-secrets.md`). Query rules for this DB: [`../docs/rds-query-rules.md`](../docs/rds-query-rules.md).

### Step 2: Start local PostgreSQL (shadow DB)

```bash
docker run -d \
  --name shadow-coordinator-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD="${SHADOW_DB_PASSWORD}" \
  -e POSTGRES_DB=shadow_rollup \
  -p 5433:5432 \
  -v shadow-coordinator-postgres-data:/var/lib/postgresql/data \
  postgres:15
```

### Step 3: Initialize the schema + import production task data

The scripted path is `scripts/00-import-bundle-range.sh` (driven by `make all`): it exports the target batches + chunks + bundles + `l2_block`/`l1_message` from production RDS and imports them, **excluding proof columns** (COMMON Trap 8), then resets proving status:

```sql
UPDATE chunk  SET proving_status = 1, total_attempts = 0, active_attempts = 0;
UPDATE batch  SET proving_status = 1, total_attempts = 0, active_attempts = 0, chunk_proofs_status = 0;
UPDATE bundle SET proving_status = 1, total_attempts = 0, active_attempts = 0;
```

Always import the **parent batch skeleton** (batch N-1 for range start N; only `state_root` must be accurate — Trap 6).

### Step 4: Populate `l2_block`

The coordinator needs `l2_block` rows linked via `chunk_hash` to format chunk tasks (Trap 49):

```bash
python3 scripts/fetch-l2-blocks.py \
  --rpc https://mainnet-rpc.scroll.io \
  --db "postgresql://…@localhost:5433/shadow_rollup" \
  --start-block <min> --end-block <max>
```

```sql
UPDATE l2_block lb SET chunk_hash = c.hash FROM chunk c
WHERE lb.number BETWEEN c.start_block_number AND c.end_block_number;
```

Note: `fetch-l2-blocks.py` cannot read withdraw roots — see Trap 56 (`batch.withdraw_root` sync).

### Step 5: Download verifier assets

```bash
VERIFIER_DIR="/tmp/shadow-verifier-assets"; mkdir -p "$VERIFIER_DIR"
# galileoV2 (v0.9.0) — from .../scroll-zkvm/releases/v0.9.0/verifier/ (see docs/CURRENT-STACK.md)
```

> ⚠️ S3 prefixes and digest encodings differ per release (v0.8.0 has no `/releases/` and Montgomery digests). Always check [`../docs/CURRENT-STACK.md`](../docs/CURRENT-STACK.md) and verify with `curl -sI` — wrong prefix = HTTP 403 (COMMON Trap 14).

### Step 6: Start coordinator

Docker or local binary; wait for `Start coordinator api successfully` (OpenVM keygen takes 2–3 min). Requires `conf/genesis.json` relative to CWD if run locally (COMMON Trap 48).

### Step 7: Start prover

```bash
./target/release/prover --config <prover.json>   # or the docker image
```

First run downloads several GB of circuit assets from S3. One work-dir per prover (Trap 51).

## Monitoring

```bash
curl -s http://localhost:8390/ | head                 # coordinator health
curl -s http://localhost:10080/health                 # prover health (per-GPU port)
psql "$SHADOW_DB" -c "SELECT proving_status, COUNT(*) FROM chunk GROUP BY proving_status;"
```

- `proving_status`: 1 = Unassigned, 2 = Assigned, 3 = Proving, 4 = Proven, 5 = Failed
- `bundle.batch_proofs_status`: 1 = Pending, 2 = Ready (relayer finalizes only at ≥ 2; `coordinator_cron` transitions it)

## Real Verifier Deployment

Two options, depending on whether your proofs share mainnet's digests.

### Option 1: Copy the mainnet verifier (only if digests match)

```bash
MAINNET_VERIFIER="0x…"; PLONK="0x…"
cast rpc anvil_setCode $MAINNET_VERIFIER $(cast code $MAINNET_VERIFIER --rpc-url <L1>) --rpc-url $ANVIL_RPC
cast rpc anvil_setCode $PLONK         $(cast code $PLONK --rpc-url <L1>)         --rpc-url $ANVIL_RPC
```

`anvil_setCode` **preserves the original immutables** (COMMON Trap 1 Cause D) — this only works when your proofs intentionally use mainnet's digests (e.g. replaying imported production proofs). For a new guest, use Option 2.

### Option 2: Deploy a fresh verifier using S3 digests

```bash
BASE_URL="https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/v0.9.0"
DIGEST1=$(curl -fsSL "${BASE_URL}/bundle/digest_1.hex" | tr -d '[:space:]')
DIGEST2=$(curl -fsSL "${BASE_URL}/bundle/digest_2.hex" | tr -d '[:space:]')

forge create --broadcast --evm-version cancun --rpc-url $ANVIL_RPC \
  --from "$OWNER" --unlocked \
  <path>/ZkEvmVerifierPostFeynman.sol:ZkEvmVerifierPostFeynman \
  --constructor-args "$PLONK_VERIFIER" "0x$DIGEST1" "0x$DIGEST2" 10
```

For v0.9.0+ the S3 digest files are canonical — use directly. v0.8.0 files are Montgomery-encoded and must be converted first ([`../docs/bundle-digest-encoding.md`](../docs/bundle-digest-encoding.md)). Sanity: `cast call $WRAPPER "verifierDigest1()(bytes32)"` must equal the S3 value.

### `finalizeBundlePostEuclidV2` uses a PostFeynman-style verifier

`ScrollChain` has exactly one bundle-finalize selector (`finalizeBundlePostEuclidV2(bytes,uint256,bytes32,bytes32,bytes)`) — the name says Euclid, but the verifier is chosen by `MultipleVersionRollupVerifier.getVerifier(version, batchIndex)` and must be a `ZkEvmVerifierPostFeynman`-style wrapper with `protocolVersion = 10` for GalileoV2 (it computes `keccak256(protocolVersion ‖ publicInput)`; the older `PostEuclid` wrapper hashes without the version prefix and rejects current proofs). Public-input layout: `chainId(8) | msgQueueHash(32) | numBatches(4) | prevStateRoot(32) | prevBatchHash(32) | postStateRoot(32) | batchHash(32) | withdrawRoot(32)`.

### Register the verifier

```bash
cast rpc anvil_impersonateAccount $OWNER --rpc-url $ANVIL_RPC
cast send $MVRV "updateVerifier(uint256,uint64,address)" 10 $START_BATCH $WRAPPER \
  --from $OWNER --rpc-url $ANVIL_RPC --unlocked
```

### Verify MVRV Routing

Always confirm routing for every target batch before finalizing (COMMON Trap 1 Cause C):

```bash
for idx in <b1> <b2> <b3>; do
  cast call $MVRV "getVerifier(uint256,uint256)(address)" 10 $idx --rpc-url $ANVIL_RPC
done
```

`anvil_setStorageAt` can override direct variables (`miscData` slot 0xa1, `nextUnfinalizedQueueIndex` slot 0x68) but **not mapping entries** — Trap 54.

## Rollup Relayer Dry-Run Mode

For testing the relayer's **calldata construction** without broadcasting: set `"dry_run": true` in the sender config. Transactions are simulated via `eth_call` (reverts still propagate), `pending_transaction` is not populated, nonce increments to mimic real behavior.

| Aspect | Verified by dry-run |
|--------|--------------------|
| Calldata encoding (ABI pack) | ✅ |
| Gas estimation + access-list path | ✅ |
| Contract revert reasons | ✅ |
| Signature / broadcast / receipt | ❌ (use a real fork) |

For a self-contained target, deploy the minimal mock ScrollChain (no-op `commitBatches`/`finalize*`) on a local Anvil and point the relayer at it; against a real fork, an `ErrorIncorrectBatchHash` on `commitBatches` is **expected** when the shadow DB is ahead of the fork block's committed state — it still proves the calldata path (see the [experiment report](../../../docs/testing_reports/snapshot-early-dryrun-and-relayer-tests.md) for what was verified). On Anvil, dry-run gas estimation may need the `estimategas.go` skip patch for blob txs.

## Relayer Finalize on the Fork (runbook)

1. **Reset rollup status** — the relayer only picks `rollup_status = 1` (`RollupPending`):

   ```sql
   UPDATE bundle SET rollup_status = 1 WHERE index BETWEEN <b1> AND <b2>;
   UPDATE batch  SET rollup_status = 1 WHERE index BETWEEN <s> AND <e>;
   ```

2. **Config essentials** (template: [`../lib/configs/relayer.json.template`](../lib/configs/relayer.json.template)): sender endpoint = Anvil RPC; **commit and finalize senders must be different addresses** (enforced at startup); authorize the commit sender as sequencer and the finalize sender as prover on the fork.
3. **Launch** with both flags: `--config <path> --min-codec-version 10`.
4. **Watch** for `Start to roll up zk proof` / `finalizeBundle in layer1` per bundle, and confirm `lastFinalizedBatchIndex` advances.

Historical first-success run (5 consecutive bundles, 2026-05): see the [report](../../../docs/testing_reports/snapshot-early-dryrun-and-relayer-tests.md).

## Known Limitations

1. **L1 messages**: chunks containing L1 messages need `scroll_getL1MessagesInBlock` RPC support; most public RPCs don't expose it. Most chunks at current mainnet height have none — usually non-blocking.
2. **Batch tasks** require all member chunks proven (`chunk_proofs_status = 2`); chunk-only testing skips this.
3. **Coordinator first start** performs OpenVM keygen (~2–3 min); first task generation after an assets swap is also slow (witness fetch) — do not restart on a quiet log.
4. **Circuit downloads** are several GB on first prover run.
5. **Orphan bundles** (historical bundle rows without imported batches) must have `batch_proofs_status = 1` or assignment deadlocks — Trap 52.
6. **`numBatches` arithmetic**: the contract computes `numBatches = batchIndex - lastFinalizedBatchIndex`; set `lastFinalizedBatchIndex` so it equals the proof's `num_batches` (single-batch bundles are easiest).
7. **Local E2E proofs cannot finalize on a mainnet fork** — different chain state (genesis, roots, message queue) → `VerificationFailed` regardless of digests.

## Common DB Fixes

```sql
-- reset proving pipeline after import
UPDATE chunk SET proving_status = 1, total_attempts = 0, active_attempts = 0;
UPDATE batch SET proving_status = 1, total_attempts = 0, active_attempts = 0, chunk_proofs_status = 0;
UPDATE bundle SET proving_status = 1, total_attempts = 0, active_attempts = 0;
```

Orphan-bundle and stale-assignment fixes: Trap 52 / Trap 53 in [`./TROUBLESHOOTING.md`](./TROUBLESHOOTING.md).

## Scripts Reference

| Script | Purpose |
|--------|---------|
| `scripts/00-import-bundle-range.sh` | Export/import a fixed bundle range from production RDS |
| `scripts/02-prepare-db.sh` | Schema migration + status resets |
| `scripts/05-wait-for-proofs.sh` / `scripts/07-wait-for-finalize.sh` | Wait for proving / finalization to complete |
| `scripts/fetch-l2-blocks.py` | Populate `l2_block` from an L2 RPC |
| `scripts/09-sync-batch-withdraw-roots.py` | Sync `batch.withdraw_root` from proof metadata (Trap 56) |
| `scripts/08-docker-orchestrate.sh` | Docker pipeline orchestration |
| `../follow/scripts/sync-mainnet-db.py` | Baseline/poll sync (follow mode; also used for one-shot baselines) |
| `../lib/01-setup-anvil.sh` … `06-run-relayer.sh` | Shared anvil/verifier/prover/relayer bring-up |
