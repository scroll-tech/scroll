# Agent Instructions for Scroll Monorepo

## Quick Orientation

This repository is a **mixed Rust + Go monorepo** for the Scroll ZK Rollup. The two most important components for proving-related work are:

- **Coordinator** (`coordinator/`) — Go service that schedules proving tasks and verifies proofs.
- **Prover** (`crates/prover-bin/`) — Rust binary that generates ZK proofs using OpenVM.
- **Shared library** (`crates/libzkp/`, `crates/libzkp_c/`) — Rust proof logic consumed by both prover and coordinator (via CGO).

For a detailed architecture overview, see [`docs/prover-coordinator-overview.md`](docs/prover-coordinator-overview.md).

## When You Are Working On an OpenVM / zkvm-prover Upgrade

Follow the structured testing guide in [`docs/openvm-upgrade-testing-guide.md`](docs/openvm-upgrade-testing-guide.md). It covers five verification levels:

1. Compilation & static checks
2. Unit tests
3. Artifact builds
4. End-to-end proving
5. Docker image builds

## Useful Commands

```bash
# Rust formatting / linting
cargo fmt --all -- --check
cargo clippy --all-features --all-targets -- -D warnings
cargo check --all-features

# Build shared library for coordinator
cargo build --release -p libzkp-c

# Build prover (CPU)
cd zkvm-prover && make prover_cpu

# Build prover (GPU)
cd zkvm-prover && make prover

# Build coordinator API
cd coordinator && make coordinator_api

# Coordinator unit tests (needs libzkp.so)
cd coordinator && make test

# E2E test setup
cd tests/prover-e2e
ln -snf <scenario> conf   # e.g., sepolia-galileoV2
make all
make coordinator_setup
```

## Directory Guide

| Directory | Purpose |
|-----------|---------|
| `crates/libzkp` | Core Rust proving/verification library |
| `crates/libzkp_c` | C FFI bindings for `libzkp` |
| `crates/prover-bin` | Prover binary (`prover`) |
| `coordinator/` | Go coordinator service |
| `rollup/` | Go rollup services (produces tasks for coordinator) |
| `tests/prover-e2e/` | E2E test harness for coordinator + prover |
| `tests/integration-test/` | General integration tests |
| `zkvm-prover/` | Build scripts and runtime config for the prover binary |
| `build/dockerfiles/` | Dockerfiles for production images |

## Coordination with Humans

- **Code / logic issues**: agents should reason independently and propose fixes.
- **Environment / secrets issues** (database passwords, RPC endpoints, cloud credentials, sudo access): ask the human and wait for a response. Do not time out and make unilateral decisions.

## Documentation Index

| Document | What It Covers |
|----------|----------------|
| [`docs/prover-coordinator-overview.md`](docs/prover-coordinator-overview.md) | Architecture, data flow, component relationships, common operations |
| [`docs/openvm-upgrade-testing-guide.md`](docs/openvm-upgrade-testing-guide.md) | Step-by-step testing checklist after OpenVM / zkvm-prover upgrades |
| [`docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md`](docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md) | Test report for PR #1783 (OpenVM 1.6.0, guest v0.8.0) |
