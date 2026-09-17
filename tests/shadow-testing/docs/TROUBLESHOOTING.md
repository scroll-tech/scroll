# Troubleshooting & Pitfalls — Trap Registry

This is the **single interface** to the trap corpus: every trap's number, its most grep-able symptom, where it lives, and its lifecycle status. Search this table first; only then read the trap body.

**Statuses**

- **Active** — live failure mode; body in its home file below.
- **Fixed** — root cause patched upstream or in the harness; body in [`TRAP-ARCHIVE.md`](TRAP-ARCHIVE.md) (kept for older builds + archaeology). Number retired, never reused.
- **Checklist** — one-time-per-machine issue absorbed into a COMMON checklist item; body in the archive.

Content files:

- [`COMMON-TROUBLESHOOTING.md`](COMMON-TROUBLESHOOTING.md) — mode-independent traps (themed groups), pre-flight ritual, checklists, top-frequency symptom table
- [`../follow/TROUBLESHOOTING.md`](../follow/TROUBLESHOOTING.md) — follow-mode traps (poll sync, relayer-on-fork, upgrades/canary, infra cost)
- [`../snapshot/TROUBLESHOOTING.md`](../snapshot/TROUBLESHOOTING.md) — snapshot-replay traps (fork state, import hygiene, pipeline)
- [`TRAP-ARCHIVE.md`](TRAP-ARCHIVE.md) — retired traps

To add a trap: take `max(existing number) + 1` across **all** files including the archive, and follow the conventions in [`../README.md`](../README.md) "Documentation Conventions" (including the new-trap bar and the trap→hardening review). `make doc-check` (from `tests/shadow-testing/`) validates numbers and references; statuses and totals in this table are maintained by hand.

## Registry (1–56)

| # | Trap | Key symptom (grep-able) | Where | Status |
|---|------|--------------------------|-------|--------|
| 1 | Wrong verifier / digest form / MVRV mis-routing / `anvil_setCode` immutables | `VerificationFailed` `0x439cc0cd` | COMMON | Active |
| 2 | Anvil forks the wrong chain (Scroll L2) | `eth_chainId 534352`; no code at ScrollChain | COMMON | Active |
| 3 | L2 RPC missing `debug_executionWitness` | chunks never assigned; debug methods blocked | COMMON | Active |
| 4 | `committedBatches` sparse (EuclidV2) | `ErrorIncorrectBatchHash 0x2a1c1442` | snapshot | Active |
| 5 | `L1MessageQueueV2` index mismatch (Sepolia) | `ErrorFinalizedIndexTooLarge/Small` | snapshot | Active |
| 6 | Parent batch missing from import | `record not found, index: <parent>` | snapshot | Active |
| 7 | Relayer nonce desync | tx sent but never mined; receipt null | ARCHIVE | Fixed |
| 8 | Production proof overwrite on import | `VerificationFailed` after import | snapshot | Active |
| 9 | Anvil `eth_estimateGas` rejects fee caps | `failed to get fee data … allowance: 0` | ARCHIVE | Fixed |
| 10 | Sender balance lost after Anvil restart | `Insufficient funds for gas * price + value` | COMMON | Active |
| 11 | Relayer started without required flags | `Required flag "min-codec-version" not set` | COMMON | Active |
| 12 | halo2 SRS not in `~/.openvm/params/` | `Params file …kzg_bn254_23.srs does not exist` | ARCHIVE | Checklist |
| 13 | Docker `--gpus device=N` + wrong `CUDA_VISIBLE_DEVICES` | `cudaErrorNoDevice`, exit 139 | ARCHIVE | Fixed |
| 14 | Verifier assets vs circuit S3 paths per release | S3 `403` downloading assets | COMMON | Active |
| 15 | Slow `l2_block` export by `chunk_hash` JOIN | export hangs, 0-byte CSV | snapshot | Active |
| 16 | Stale proofs after source revert/restore | `mismatch batch-proof exe commitment` | COMMON | Active |
| 17 | `coordinator_cron` re-marks corrupt bundles ready | `unexpected end of JSON input` flood | COMMON | Active |
| 18 | Relayer creates competing batches | provers busy, target bundles stall | snapshot | Active |
| 19 | `lastCommittedBatchIndex` desync | `ErrorBatchNotCommitted 0x227a699e` | COMMON | Active |
| 20 | `nextCrossDomainMessageIndex` too low | `0x16465978` (`\x16FYx`) | COMMON | Active |
| 21 | Stale cached proof in prover local DB | `Invalid VData … does not match number of AIRs` | COMMON | Active |
| 22 | Poll-sync `l2_block` linkage / starvation / NULL parent links | `failed to fetch block hashes of a chunk` | follow | Active |
| 23 | Bundles popping post-fork L1 messages | `VerificationFailed` then `0x16465978` (L1-msg bundles) | follow | Active |
| 24 | Collection timeouts < real proof times | `proof task have reach the timeout` | COMMON | Active |
| 25 | Orphaned poll-sync backfills full history | `inserted 25000/6642569` | ARCHIVE | Fixed |
| 26 | Relayer L2 watcher genesis-crawls empty `l2_block` | `l2_block` fills from block 1 | ARCHIVE | Fixed |
| 27 | Relayer cannot commit/finalize — balance/sequencer/nonce | `allowance: 0` / `ErrorCallerIsNotSequencer` / tx queued forever | follow | Active |
| 28 | Post-Fusaka blob base-fee explosion | `Insufficient funds`; blob-base-fee ≈ 1e18 | ARCHIVE | Fixed |
| 29 | Same-block `addProver` ordering | `ErrorCallerIsNotProver 0x7b263b17`; `rollup_status=7` | ARCHIVE | Fixed |
| 30 | Fresh shadow DB schema never migrated | `relation "public.<table>" does not exist` | ARCHIVE | Checklist |
| 31 | Upgrade Phase 1 built from branch, not production | Phase-1 finalizes all `VerificationFailed` | follow | Active |
| 32 | `cargo update` drifts revm / `[patch]` resolution | `E0308 … multiple versions of revm_primitives` | follow | Active |
| 33 | v0.9.0+ requires `agg_vk.bin` on S3 + in assets | `agg_vk.bin 403` / `missing from assets` | follow | Active |
| 34 | Stale `prover_task` failure rows starve assignment | `chunk_proofs_status=2` forever, no ERROR | follow | Active |
| 35 | Post-upgrade old-format proofs poison dispatch | `untagged enum ProofEnum` loop | follow | Active |
| 36 | `cast receipt status` output drift | `updateVerifier reverted (status 'true')` | ARCHIVE | Fixed |
| 37 | `make coordinator_api` doesn't refresh `libzkp.so` | `Batch verify failed … missing from assets` (stale .so) | follow | Active |
| 38 | halo2-gpu VRAM starvation / `def_hook_commit` | `cudaErrorInvalidConfiguration quotient.cu` | follow | Active |
| 39 | Canary proof-import pitfalls | `bundle.proof` stays NULL; quarantine grows | follow | Active |
| 40 | Canary boundary N vs proven backlog | old-proof backlog finalize-fails post-t1 | follow | Active |
| 41 | Remote queries missing `deleted_at IS NULL` | RDS bill spike; `DataFileRead` minutes/cycle | follow | Active |
| 42 | Docker prover: CPU-SNARK images / assignment rows | bundle 30+ min; `already assigned a task` | follow | Active |
| 43 | `11-follow-stop.sh` aborts midway | `make follow-stop` Error 2, half stack up | ARCHIVE | Fixed |
| 44 | Anvil silent death / stall — `--state` relaunch | RPC refused / anvil hang mid-run | COMMON | Active |
| 45 | Tx landed on-chain but never recorded | `commitBatch` revert spam / bundle stuck `rs1` vs chain advanced | COMMON | Active |
| 46 | `cast code` returns `0x` for codeless accounts | "copied" verifier has no code | ARCHIVE | Fixed |
| 47 | `set -euo pipefail` + failing `$( )` | script dies after step header, no error | ARCHIVE | Fixed |
| 48 | `coordinator_api` genesis.json relative to CWD | `failed to read genesis` seconds after start | ARCHIVE | Fixed |
| 49 | Chunk data prerequisites for task assignment | prover polls empty; coordinator healthy | snapshot | Active |
| 50 | Fork block / codec version mismatch | `mismatched post-state root` | snapshot | Active |
| 51 | Multiple provers sharing one circuit cache | `File exists (os error 17)` | ARCHIVE | Fixed |
| 52 | Orphan bundles deadlock assignment | no bundle task ever assigned | snapshot | Active |
| 53 | Imported `status=2` + NULL proof rows | `chunk_proofs_status=2` then format fails | snapshot | Active |
| 54 | `anvil_setStorageAt` ignored on mapping slots | patched storage visible but `eth_call` uses old | snapshot | Active |
| 55 | Anvil default account not an EOA in fork mode | `ErrorAccountIsNotEOA` on `addProver` | snapshot | Active |
| 56 | Stale `batch.withdraw_root` in finalize | `VerificationFailed` w/ correct digests+routing | snapshot | Active |

**Totals**: 41 Active · 13 Fixed · 2 Checklist. Next free number: **57**.
