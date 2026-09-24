# Snapshot Mode — Early Dry-Run & Relayer Experiments (postmortem, 2026-05…07 era)

> Moved out of `snapshot/GUIDE.md` and `docs/COMMON-TROUBLESHOOTING.md` on 2026-09-11 so the GUIDE stays a runbook. Content is a historical record — addresses, batch numbers and digests below were per-experiment; do not reuse them. Current facts live in [`../tests/shadow-testing/docs/CURRENT-STACK.md`](../tests/shadow-testing/docs/CURRENT-STACK.md).
>
> Era: Anvil fork blocks ~25.20–25.21M, bundles 17297–17301 / 17330, batches ~517.7–517.8k. Guest under test: v0.9.0 pre-release.

## 1. Dry-run against a Mock ScrollChain (mock contract)

| Transaction | Status | Notes |
|-------------|--------|-------|
| `commitBatches` | ✅ `eth_call` succeeded | Selector `0x9bbaa2ba` via mock `commitBatches(uint8,bytes32,bytes32)` |
| `finalizeBundlePostEuclidV2NoProof` | ✅ `eth_call` succeeded | Selector `0xbd6f916b` via mock no-op |
| `finalizeBundlePostEuclidV2` (with proof) | ✅ `eth_call` succeeded | Bundle 17301 with valid `OpenVMBundleProof` |

## 2. Dry-run against the real ScrollChain proxy (fork block 25206000)

| Transaction | Status | Notes |
|-------------|--------|-------|
| `commitBatches` | ⚠️ reached contract, reverted `ErrorIncorrectBatchHash` | Shadow DB batches (518565+) were ahead of the fork block's committed state (~517816) — expected, confirms calldata path |
| `finalizeBundlePostEuclidV2` | ✅ succeeded | Bundle 17330 (batch 517809), real mainnet proof — see §3 |

Per-run deployed contracts (fork-only, now stale):

| Contract | Address | Notes |
|----------|---------|-------|
| ZkEvmVerifierPostFeynman (v10) | `0x0dE180164Dc571522457101F5c47B2eaB36d0A82` | copied from mainnet via `anvil_setCode` — only valid for mainnet-digest proofs |
| Plonk verifier (v10) | `0x749fc77a1a131632a8b88e8703e489557660c75e` | copied from mainnet |
| ZkEvmVerifierPostFeynman (wrong) | `0xc3230A4C89a5Ce0455414215e533de4D8849b3f8` | deployed with wrong (Montgomery) digests — do not use |

## 3. End-to-end `finalizeBundlePostEuclidV2` dry-run success (bundle 17330)

- Batch 517809, codec V10, single-batch; proof replayed from the real mainnet finalize tx `0x753f8f9ca01d4e67f710c6dab8ce0b17a17a7ad46a9d7480d92657803a36ca24`.
- Public-input hash recomputed (`protocolVersion ‖ 204-byte publicInput`, 236 bytes total keccak) matched `bundle_pi_hash` exactly: `0xcd4421bad526bd108d9ae8c2af3d46ea1a986207f0b8c1af781b601c1ae50e5a`.
- Fork state resets applied pre-execution: `miscData` slot 0xa1 (`lastFinalized=517808`), `nextUnfinalizedQueueIndex` slot 0x68 → 0, `addProver` with a fresh EOA (Anvil default account carries 7702 code → `ErrorAccountIsNotEOA`, now Trap 55).
- Result: tx success, 425,719 gas, `lastFinalizedBatchIndex` → 517809, `finalizedStateRoots[517809]` set, queue index advanced to 998288.
- `anvil_setStorageAt` mapping-slot discovery (now Trap 54): direct-variable slots patch fine; mapping entries are ignored by the EVM.

## 4. Multi-bundle relayer finalize test (5 consecutive bundles, first success)

Bundles 17297–17301 (batches 517761–517765), shadow proofs, wrapper `0xb1F2C5c1ea2885278a1070350d12d3D8824265B0` (a per-run deploy):

| Bundle | Batch | Tx | Status | Gas |
|--------|-------|----|--------|-----|
| 17297 | 517761 | `0x6d6264…cdaa725` | ✅ | 439,987 |
| 17298 | 517762 | `0x071268…1136516` | ✅ | 407,455 |
| 17299 | 517763 | `0x8f8894…6cabd5c` | ✅ | 407,479 |
| 17300 | 517764 | `0xa87721…302cd3e` | ✅ | 407,419 |
| 17301 | 517765 | `0x41ee42…c9cf891` | ✅ | 401,404 |

All five finalized consecutively with no manual intervention; `lastFinalizedBatchIndex` 517760 → 517765. Pre-conditions that mattered (now traps): reset `bundle`/`batch.rollup_status = 1` (relayer only picks `RollupPending`), commit/finalize senders must be distinct addresses, and the finalizer loop runs independently of the (failing) batch committer.

## 5. Source-hygiene lesson (from the v0.9.0 multi-bundle test)

Do not `git checkout --` uncommitted source changes blindly: the v0.9.0 adaptations (`libzkp`, `prover-bin`, Go message types, `rust-toolchain`, `Cargo.lock`) were once wiped this way and had to be reconstructed from compiler errors. Check `git diff --stat` before bulk reverts; stage or stash anything you intend to keep.
