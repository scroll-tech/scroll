# OpenVM Upgrade Testing Guide

This document describes the verification steps required after bumping the zkvm-prover / OpenVM version in the Scroll monorepo.

## Context

The prover relies on external `zkvm-prover` and `scroll-zkvm-*` crates pinned to specific git revisions. When OpenVM is upgraded (e.g., `1.4.2 → 1.4.3`), the following typically change:

- `Cargo.toml` / `Cargo.lock` dependency revisions
- Rust toolchain version (OpenVM often requires a specific nightly)
- Circuit parameters (`segment_len`, memory limits, etc.)
- Witness serialization format
- VK (verification key) assets

Because the coordinator links `libzkp.so` (which also depends on these crates), both sides must be rebuilt and tested together.

## Pre-Test Checklist

Before starting the full test suite, verify:

1. [ ] `rust-toolchain` matches the version expected by the new OpenVM release.
2. [ ] `Cargo.toml` git revisions for `scroll-zkvm-prover`, `scroll-zkvm-verifier`, `scroll-zkvm-types` are correct.
3. [ ] Coordinator's `conf/config.json` (or template) lists the expected fork names and does not reference deprecated features (`legacy_witness`, `openvm_13`).
4. [ ] E2E test configs (`tests/prover-e2e/*/config.json`) use the correct `SCROLL_ZKVM_VERSION` and `SCROLL_FORK_NAME`.
5. [ ] Guest asset version matches the prover's OpenVM version. Mismatch causes `Invalid app vm commit` during proving/verification. If the guest assets were compiled with a different OpenVM version, they must be recompiled and re-uploaded.
6. [ ] `RUST_MIN_STACK` is exported (e.g., `export RUST_MIN_STACK=16777216`). Batch and bundle proving can exceed the default thread stack size, causing silent stack overflow crashes without this setting.

## Test Levels

### Level 1 — Compilation & Static Checks
**Goal:** Ensure the code compiles and passes lint.

```bash
# Rust
cargo fmt --all -- --check
cargo clippy --all-features --all-targets -- -D warnings
cargo check --all-features

# Go (coordinator & rollup)
cd coordinator && go build ./...
cd rollup && go build ./...
```

**What failures indicate:**
- Dependency resolution errors (wrong git rev, conflicting versions)
- API breakage between OpenVM versions (type mismatches, removed methods)
- Missing feature flags

### Level 2 — Unit Tests
**Goal:** Validate core logic without full proving.

```bash
# Rust
cargo test -p libzkp
cargo test -p prover-bin

# Go (coordinator; requires libzkp.so)
cd coordinator
make libzkp
make test
```

**Focus areas after an OpenVM upgrade:**
- Witness encoding / decoding (`tasks/chunk.rs`, `tasks/batch.rs`, `tasks/bundle.rs`)
- Proof verification paths (`proofs.rs`)
- Version parsing (`coordinator/internal/utils/version.go`)

### Level 3 — Artifact Build
**Goal:** Produce the final binaries and shared libraries.

| Artifact | Command | Consumer |
|----------|---------|----------|
| `libzkp.so` | `cargo build --release -p libzkp-c` | Coordinator (CGO) |
| `prover` | `cd zkvm-prover && make prover_cpu` (or `make prover` for GPU) | Prover node |
| `coordinator_api` | `cd coordinator && make coordinator_api` | Coordinator node |
| `e2e_tool` | `cd tests/prover-e2e && make test_tool` | E2E harness |

**Check:** Run each binary with `--version` and confirm the reported ZK version matches expectations.

**Guest asset version compatibility:** After building `prover`, verify that the guest assets (VKs, app config, ELF files) were compiled with the same OpenVM version the prover links against. The simplest check is to run a chunk proving task and confirm the prover does not panic with `Invalid app vm commit`. If it does, the guest assets must be recompiled with the matching OpenVM version and re-uploaded.

### Level 4 — End-to-End Proving
**Goal:** Run the full coordinator → prover → coordinator loop with real task data.

Recommended scenario for GalileoV2-related upgrades: `sepolia-galileoV2`.

```bash
cd tests/prover-e2e
ln -snf sepolia-galileoV2 conf
make all              # Start DB + import blocks
make coordinator_setup
```

Then:
1. Launch `coordinator_api` with the generated `build/bin/conf/config.json`.
2. In `zkvm-prover/`, create `config.json` from `config.template.json` and set `sdk_config.coordinator.base_url` to `http://localhost:8390`.
3. Run the prover:
   ```bash
   cd zkvm-prover
   make test_e2e_run        # CPU
   # or
   make test_e2e_run_gpu    # GPU
   ```

**What to watch:**
- Prover successfully downloads assets for the target fork.
- Chunk tasks are picked up and proved without panic.
- Batch tasks aggregate chunk proofs correctly.
- Bundle tasks aggregate batch proofs correctly.
- Coordinator verifies all returned proofs and marks tasks `success`.
- Memory usage stays within node limits (OpenVM upgrades may change `segment_len`, affecting RAM).
- **GPU timeout:** On GPU-enabled provers, batch and bundle proving can take 1–3 minutes each. Ensure `chunk_collection_time_sec` (coordinator config) is set high enough (e.g., `3600`) so the coordinator does not abort long-running sessions. The default `180` is too short for GPU batch/bundle proving.

### Level 5 — Docker Image Build
**Goal:** Ensure production images build correctly.

```bash
cd coordinator
make docker
```

**Common issues after upgrades:**
- Dockerfile base image (`scrolltech/go-rust-builder`) uses an older Rust nightly than `rust-toolchain`.
- Missing `riscv32im-unknown-none-elf` target in the Docker build stage.
- CGO linker flags incompatible with the new `libzkp.so`.

## Special Test Cases

### Dump Command
After adding or modifying the `dump` functionality:

```bash
../target/release/prover --config config.json dump chunk <task_id>
../target/release/prover --config config.json dump --json chunk <task_id>
```

Verify:
- Files are written to the expected path.
- `--json` produces readable JSON; without it produces `input_task.bin` and `agg_proofs.bin`.
- The command exits with a non-zero status (dump intentionally returns `TaskStatus::Failed` to halt the service loop).

### Debug Mode
Enable `debug_mode` in the prover config and verify that existing local assets are reused without HEAD requests.

When `debug_mode` is `true`, the prover skips re-downloading assets that already exist locally and does not send HEAD requests to the remote server. This speeds up local development and E2E testing significantly, but it also means stale local assets will not be refreshed automatically. Always clear `.work/<fork>/` when switching between asset versions.

**Usage in local E2E:**
```json
{
  "circuits": {
    "galileo": {
      "debug_mode": true
    }
  }
}
```

If you see `Invalid app vm commit` or unexpected verification failures with `debug_mode: true`, delete the local workspace (e.g., `.work/galileo/`) to force a fresh asset download.

### Fork Compatibility
If the PR adds support for a new fork (e.g., `galileoV2`):
- Confirm the coordinator config contains the fork in the verifier list.
- Confirm the prover can download the corresponding VK asset.
- Run an E2E scenario that targets that fork.

## Sign-Off Criteria

An OpenVM upgrade PR can be considered fully tested when:

1. All five levels above pass without errors.
2. At least one E2E scenario completes the full chunk → batch → bundle pipeline.
3. Docker images build and the container starts (`coordinator_api --version` succeeds inside the image).
4. No new Clippy warnings or formatting regressions.
5. Coordinator config does not reference deprecated features (`legacy_witness`, `openvm_13`).
