# Multi-GPU Prover Support: Investigation & Plan

Date: 2026-09-24. Status: research only, no code changes made.

Question: does `crates/prover-bin` support multiple GPUs, and if not, what would each component need to change? Also catalogs legacy designs that are candidates for removal (proving-sdk cloud/local split, coordinator auth, etc.).

## TL;DR

- The prover does **not** support multi-GPU, and has **no GPU configuration option at all**. Device selection is implicit ("current CUDA device", i.e. device 0) all the way down; the only selector is the out-of-process `CUDA_VISIBLE_DEVICES` / docker `NVIDIA_VISIBLE_DEVICES` pinning we already use.
- A single proof (chunk/batch/bundle) is strictly single-GPU; there is no multi-GPU MSM/NTT anywhere in the stack.
- A single prover process runs exactly one task at a time, enforced at three layers: SDK (`n_workers > 1` is rejected for local provers), `LocalProver.current_task: Option<JoinHandle>`, and coordinator (one assigned task per public key).
- **Pragmatic recommendation**: the current "one container per GPU" deployment has identical throughput, zero code cost, and better fault isolation. Single-process multi-GPU is blocked by global singletons in two upstream crates (halo2-gpu, openvm cuda-common) and is not worth forking them for.
- What *is* worth doing first (and unlocks any future multi-GPU work): vendor the proving SDK into this repo, make prover-bin multi-task, and simplify coordinator auth while keeping identity/version-routing.

## 1. Current GPU usage

### Device selection: implicit, unconfigurable

- `zkvm-prover/config.json.template` contains no GPU/device field. The only related knobs are `sdk_config.prover.supported_proof_types`, `circuit_version`, and the SDK's `n_workers` (default 1, env `N_WORKERS`, rejected for local provers).
- OpenVM STARK CUDA backend: `GpuDevice::new` uses `get_device()` (the current device) then `GpuDeviceCtx::for_device(id)` — `stark-backend/crates/cuda-backend/src/device.rs:64`, `cuda-common/src/stream.rs:146-157`.
- halo2-gpu (bundle SNARK stage): crate-level lazy singleton bound to whatever device is current at first touch — `halo2-gpu/halo2_proofs/src/cuda/utils.rs:33-35`: `pub static HALO2_GPU_CTX: Lazy<GpuDeviceCtx>`.
- `set_device_by_id` exists (`cuda-common/src/common.rs:31`) and halo2-gpu even pins worker threads back to `HALO2_GPU_CTX.device_id` before each FFI (`utils.rs:144-147`), but no upper-layer code ever passes a non-default device.
- Zero code references to `CUDA_VISIBLE_DEVICES` in the whole dependency tree; only docs mention it (`tests/prover-e2e/README.md:55`). The CUDA runtime interprets it and remaps the pinned physical card to in-container device 0 — that is the entire multi-GPU mechanism today.

### Single-proof parallelism: strictly one card

- STARK stage: all kernels run on one `GpuDeviceCtx` (one non-blocking stream). NTT twiddle caches are keyed by `(device_id, reset_epoch)` (`cuda-backend/src/cuda/batch_ntt_small.rs:52-56`) — upstream anticipated in-process multi-device, but the proving pipeline itself never shards across cards.
- halo2 SNARK stage: everything goes through the `HALO2_GPU_CTX` singleton; function names say it outright (`poly_multiply_add_single_gpu`, `halo2_proofs/src/poly.rs:846`). No multi-GPU MSM/NTT.

### Task concurrency: explicitly forbidden

- SDK hard ban: `scroll-proving-sdk/src/prover/builder.rs:33-35` — `if self.proving_service.is_local() && self.cfg.prover.n_workers > 1 { bail!(...) }`.
- Single task slot: `LocalProver.current_task: Option<JoinHandle<Result<String>>>` (`crates/prover-bin/src/prover.rs:206`); `query_task` ignores `req.task_id` and only inspects this slot (prover.rs:234-235); `do_prove` overwrites it (prover.rs:459).
- Handler-level mutual exclusion: `handlers: HashMap<vk, Arc<Mutex<UniversalHandler>>>` held for the whole proof (prover.rs:445-458), with double/triple lock ordering for deferral init (prover.rs:404-415).
- Runtime: `#[tokio::main(flavor = "current_thread")]` (main.rs:75); proving offloaded via `spawn_blocking`.
- VRAM: the VPMM memory pool never returns physical pages (`universal.rs:39-47`); each circuit SDK lazily uploads GiBs of proving keys (~5.7 GiB just to construct the agg prover for VKs, `universal.rs:54-56`); the halo2-gpu SNARK needs ~24 GB VRAM. `UniversalHandler::reset()` exists to free child-circuit GPU keys before a bundle — a single card is already tight, so per-card task concurrency is pointless anyway.

### Dependency versions (Cargo.lock)

- `scroll-proving-sdk` v0.3.0, git rev `4266daa` (`crates/prover-bin/Cargo.toml:12`)
- `scroll-zkvm-prover` v0.9.0 = zkvm-prover repo rev `bf887150`
- `halo2-axiom-gpu` v1.0.0 via openvm-sdk `halo2-gpu` feature → snark-verifier-sdk `cuda` → halo2-base `cuda`

## 2. Change list for single-process multi-GPU

Target: one prover process using N cards, one independent task per card.

### A. prover-bin (this repo) — small to medium

1. Add `gpu_devices: Vec<u32>` to `LocalProverConfig` (`crates/prover-bin/src/prover.rs:167-171`). [small]
2. `current_task: Option<...>` → `HashMap<String, JoinHandle>`; make `query_task` look up by `req.task_id` (prover.rs:203-209, 234-280). [medium]
3. Handler cache key `vk` → `(vk, device_id)`: one `UniversalHandler` set per card, proving keys cannot cross devices (prover.rs:474-501). [medium]
4. Task→device assignment in `do_prove` (prover.rs:324): pick a free device, call `openvm_cuda_common::common::set_device_by_id(dev)` at the top of the `spawn_blocking` thread, before touching any lazy singleton. [small, depends on C/D]
5. Optionally switch main.rs:75 to a multi-thread runtime (not strictly needed; the blocking pool suffices). [small]
6. Revisit the "task panic → exit whole process" policy (prover.rs:250-255): under single-process multi-GPU it amplifies the blast radius.

### B. scroll-proving-sdk (external git dep — fork or vendor first) — small

7. Remove/relax the `builder.rs:33` `is_local() && n_workers > 1` ban so `n_workers` can match GPU count. [small]
8. `ProvingService` is wrapped in an `RwLock` and both `prove`/`query_task` take the **write** lock (`prover/mod.rs`), so even multiple workers would serialize — must be split. [small]
9. Verify `Db` per-worker keying is concurrency-safe (mod.rs:154-167; already per-key). [verify]

### C. halo2-gpu (axiom-crypto/halo2-gpu v1.0.0, needs a fork) — large, the hard blocker

10. Turn `HALO2_GPU_CTX` (`cuda/utils.rs:33`) into a per-device structure (`Lazy<RwLock<HashMap<u32, GpuDeviceCtx>>>` or thread-local) and update the dozens of `DeviceBuffer::with_capacity_on(&HALO2_GPU_CTX)` call sites (evaluation.rs, modules.rs, …) to resolve the ctx from the current thread's device. `ensure_current_device_matches_ctx` (utils.rs:144) semantics change accordingly. [large]

### D. openvm cuda-common / cuda-backend (openvm-org/stark-backend v2.0.0, needs a fork) — medium to large

11. `MEMORY_MANAGER` (`cuda-common/src/memory_manager/mod.rs:30`) is a process-wide VPMM singleton; must be partitioned per device or cross-card allocations corrupt each other. [large]
12. NTT twiddles are already keyed by device (batch_ntt_small.rs:52) and `GpuDevice`/`GpuDeviceCtx` are parameterized — the STARK side is mostly verification work. [verify]

### E. Engineering / ops — small

13. VRAM quota: one proof task per card at a time (24 GB-class VRAM + never-returned VPMM pages make same-card concurrency useless). Watch deferral-init lock ordering across cards. [medium]
14. Deployment: single-container multi-card would drop `NVIDIA_VISIBLE_DEVICES` pinning in favor of the new `gpu_devices` config (`build/dockerfiles/prover.Dockerfile`). [small]

## 3. Coordinator side

### Task dispatch model today

- Identity is the **public key** (derived from a random key file in `keys_dir`), not `prover_name`. `prover_name` has no uniqueness constraint.
- One public key holds at most one assigned task: `IsProverAssigned(public_key)` (`coordinator/internal/orm/prover_task.go:60-71`). Same-type re-request returns the same task (idempotent re-fetch); cross-type returns "already assigned a task".
- Submission is located by UUID + public key (`logic/submitproof/proof_receiver.go:158`), so it is naturally multi-task-ready. Timeout sweeper (`controller/cron/collect_proof.go`) resets per-uuid — also naturally compatible.
- `prover_task` has no unique constraint on public key — **no DB migration needed** for multi-slot.

### Options

- **Option A (zero coordinator change, recommended)**: one prover process + one key file per GPU worker. Each gets its own public key and task slot. This is exactly today's multi-container pattern and already works.
- **Option B (single identity, N task slots)**: rework `checkParameter` (`logic/provertask/prover_task.go:147-200`) from single-slot to "free slot available" (config `max_tasks_per_prover`); rewrite the three isomorphic "already assigned" branches in `chunk/batch/bundle_prover_task.go` to distinguish idempotent re-fetch (by uuid) from new-task requests. ~2-4 days including tests, but the larger cost is on the Rust side (multi-task prover-bin + SDK).

## 4. Coordinator auth: what to keep, what to drop

Current flow: `GET /challenge` → JWT with random 32-byte challenge → `POST /login` with RLP-encoded, secp256k1-signed login message (verified in `coordinator/internal/types/auth.go:83-104`) → login JWT → `loginMiddleware` on `/get_task`, `/submit_proof`. Challenges are inserted into the `challenge` table for replay protection, cleaned by `cron/cleanup_challenge.go`.

Important: **public key is not just auth, it is the task-ownership primary key** (prover_task rows, block list, timeout reset all key on it). And `CompatiblityCheck` / `ProverHardForkName` (`logic/auth/login.go:85-146`) are version-routing logic, not security — they must stay.

Recommended path (zero SDK change): keep the `/login` interface shape but skip challenge/signature verification and issue a long-lived token (or read identity from the request body). Deleting auth entirely ("Option B: no identity") is **not viable** — one shared key collapses the cluster to one task at a time.

Change surface: `route/route.go:29-41` (drop `/challenge`, middleware), `controller/api/auth.go:52-69`, `logic/auth/login.go` (drop `VerifyMsg`/dedup, keep compatibility checks), `orm/challenge.go` + cleanup cron (delete; optional drop-table migration), `config` auth section, `controller/proxy/*` if the proxy is retired too, plus mock prover / api tests.

## 5. Legacy design cleanup candidates

### scroll-proving-sdk (~1680 lines; ~1500 used, ~600+ droppable)

- Cloud prover is already 95% pruned: only `is_local()`, `format_cloud_prover_name` (`utils.rs:32-35`), and `ProverProviderType::Internal/External` remain; `examples/cloud.rs` is all `todo!()`; `builder.rs:52` hardcodes `Internal` with a FIXME about a coordinator external-prover bug.
- `prover-bin` is the **only** consumer in the workspace (3 files: main.rs, prover.rs, dumper.rs). Cloud paths are unreachable from it.
- Vendor-then-prune is safer than rewrite (keeps the RLP login signature byte-compatible; there are compatibility tests to copy). A from-scratch minimal client is ~800-1000 lines; Go-side protocol references exist in `coordinator/internal/controller/proxy/client.go:72-224` and `coordinator/test/mock_prover.go`.
- Vendoring also drops heavy deps: rocksdb, the ethers-core git fork (key_signer → k256/alloy-signer, ~50 lines), axum (health endpoint → a dozen lines of hyper or plain TCP), reqwest-middleware/reqwest-retry.
- SDK `build.rs` requires compile-time env vars `GO_TAG`/`GIT_REV`/`ZK_VERSION` (injected by `zkvm-prover/Makefile:25-34`) — a real coupling pain that vendoring removes.

### prover-bin

- `src/types.rs`: whole file is `#![allow(dead_code)]`, unreferenced duplicates of SDK types.
- `zk_circuits_handler.rs:1`: commented-out `//pub mod euclid;` remnant.
- `LocalProver::get_vks` deprecated, returns empty (prover.rs:216-222).
- `use_openvm_13` path (`libzkp/src/tasks.rs:43`, prover.rs:330-332): placeholder that just bails.

### coordinator

- `prover_task.reward` column (`orm/prover_task.go:38`): reward mechanism leftover, never written.
- `ProverTypeChunkDeprecated`/`ProverTypeBatchDeprecated` (`types/prover.go:35-44`) and the `prover_types` login field: legacy, only OpenVM remains.
- VK consistency check (`logic/auth/login.go:96-110`): self-described "code only for backward compatibility"; SDK `get_vks` returns empty.
- `GetTaskSchema.UseSnark` (`types/get_task.go:16`): halo2-era leftover.
- proxy `compatibileMode` (`controller/proxy/client.go:140-165`): forges signatures for old coordinators.
- `TaskTimeoutMoreThanOnce` (`orm/prover_task.go:205-206`): self-described "a temp design".
- `prover_block_list` + external prover branches (`ProverProviderTypeExternal`, `ExternalProverThreshold`, `cloud_prover_{provider}_index` naming, `batch_prover_task.go:70-80`, `utils/prover_name.go`): external provers no longer connect, and a blocked prover can regenerate its key file to evade the block anyway — retire together with auth simplification.

## 6. Recommended sequencing

1. **Vendor the proving SDK** into this repo and prune cloud/dead code. Unblocks everything else and removes the external-repo release coupling.
2. **prover-bin multi-task** (`current_task` → map, per-task query).
3. **Coordinator auth simplification**: keep interface shape + identity + version/fork routing; drop challenge/signature verification; retire challenge table, block list, external-prover branches.
4. Only if single-process multi-GPU is still wanted after that: fork halo2-gpu + openvm cuda-common singletons (section 2 C/D). Until then, one container per GPU remains the production answer.
