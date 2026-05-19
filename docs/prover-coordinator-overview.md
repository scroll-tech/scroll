# Prover-Coordinator System Overview

## Architecture

The Scroll proving stack consists of two primary runtime components written in Rust and Go, plus a shared Rust library used for proof verification.

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Coordinator (Go)                            │
│                    Task scheduler & proof verifier                   │
│  • Reads pending chunk / batch / bundle tasks from PostgreSQL       │
│  • Distributes tasks to connected provers                           │
│  • Validates proofs submitted by provers via libzkp (FFI)           │
│  • Serves gRPC/HTTP APIs for provers and rollup services            │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ assign / submit
┌──────────────────────────────▼──────────────────────────────────────┐
│                      Prover (Rust, prover-bin)                      │
│                     ZK proof generator                              │
│  • Polls coordinator for proving tasks                              │
│  • Executes zkvm-prover (OpenVM) to generate STARK/SNARK proofs     │
│  • Submits proof results back to coordinator                        │
│  • Supports three task types: Chunk, Batch, Bundle                  │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Rollup services** (in `rollup/`) ingest L2 blocks, group them into **chunks**, **batches**, and **bundles**, and write task records into PostgreSQL.
2. **Coordinator** (`coordinator/`) discovers these records and advertises them to connected provers.
3. **Prover** (`crates/prover-bin/`) fetches a task, downloads the required verification key (VK) asset, runs the zkVM circuit, and produces a proof.
4. **Coordinator** receives the proof and calls `libzkp` (via CGO) to verify it before marking the task complete.

## Key Components

### 1. `crates/libzkp` & `crates/libzkp_c`

Shared Rust library containing:
- Witness serialization / deserialization for chunk, batch, and bundle tasks
- Proof verification logic
- FFI bindings (`libzkp_c`) so the Go coordinator can link against `libzkp.so`

**Important:** Any breaking change to witness formats or proof structures requires rebuilding `libzkp.so` and relinking the coordinator.

### 2. `crates/prover-bin`

The prover binary. Key concepts:
- **UniversalHandler**: Dynamically loads circuit assets (VKs, params) based on the task's fork name (e.g., `galileo`, `galileoV2`).
- **AssetsLocationData**: Configures where to download circuit assets from. `debug_mode` skips remote size verification when a local file already exists.
- **Task types**:
  - `Chunk`: Prove a group of L2 blocks
  - `Batch`: Prove a group of chunks
  - `Bundle`: Prove a group of batches
- **Commands**:
  - `handle <json>`: Prove a specific set of task IDs
  - `dump <type> <id>`: Export task input data to local files (JSON or bincode) without proving
  - (no subcommand): Run as a persistent service polling coordinator

### 3. `coordinator/`

Go service with three entry points:
- `coordinator_api`: Main API server
- `coordinator_cron`: Background jobs
- `coordinator_tool`: CLI utilities

The coordinator links against `libzkp.so` at build time. The `localsetup` Makefile target builds the `.so`, builds the binary, and copies config templates.

### 4. `tests/prover-e2e`

End-to-end test harness for the proving pipeline.

- Each subdirectory (`sepolia-galileoV2`, `mainnet-galileo`, `cloak-galileoV2`, etc.) contains:
  - `.make.env`: Block range and fork name
  - `config.json`: Database / RPC config for the test tool
  - `config.template.json`: Coordinator config template
  - `genesis.json`: Chain genesis for the local import
  - `.sql` files (optional): Pre-seeded block data

- **Workflow**:
  1. `make all` → starts PostgreSQL container, builds `e2e_tool`, imports blocks from RPC into DB
  2. `make coordinator_setup` → builds coordinator API binary and copies config
  3. Run `coordinator_api` locally
  4. Configure and run `prover` → connects to local coordinator and proves injected tasks

## Versioning & Compatibility

- **Workspace version**: defined in root `Cargo.toml` (`workspace.package.version`)
- **ZK version**: derived from `zkvm-prover` git commit + `Plonky3` git commit, embedded into coordinator/prover binaries at build time
- **Fork names**: hard-fork identifiers (`galileo`, `galileoV2`, ...) that select which circuit assets and witness formats to use

Prover and coordinator must agree on:
1. ZK version (assets compatibility)
2. Fork name support (coordinator's verifier config must list the fork, prover must have the asset)
3. Task type enum values

## Common Operations

### Build prover (CPU)
```bash
cd zkvm-prover
make prover_cpu
# Binary appears at ../target/release/prover
```

### Build prover (GPU / CUDA)
```bash
cd zkvm-prover
make prover
```

### Build coordinator
```bash
cd coordinator
make coordinator_api
# Requires libzkp.so to be built first
```

### Run local E2E test
```bash
cd tests/prover-e2e
ln -s sepolia-galileoV2 conf
make all
make coordinator_setup
cd ../../coordinator/build/bin
# edit conf/config.json if needed
./coordinator_api
# In another terminal:
cd zkvm-prover
# copy config.template.json → config.json, set coordinator base_url
make test_e2e_run
```

## Notes for Operators

- The prover downloads large circuit assets (VKs, params) on first use. Ensure sufficient disk space and network bandwidth.
- `debug_mode` in `AssetsLocationData` is intended for local development only; do not enable in production.
- A prover panic now triggers `resume_unwind`, killing the worker thread (and potentially the process). Monitor prover restarts in production.
