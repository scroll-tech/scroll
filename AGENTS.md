# Agent Instructions for Scroll Monorepo

## Quick Orientation

This repository is a **mixed Rust + Go monorepo** for the Scroll ZK Rollup. The two most important components for proving-related work are:

- **Coordinator** (`coordinator/`) — Go service that schedules proving tasks and verifies proofs.
- **Prover** (`crates/prover-bin/`) — Rust binary that generates ZK proofs using OpenVM.
- **Shared library** (`crates/libzkp/`, `crates/libzkp_c/`) — Rust proof logic consumed by both prover and coordinator (via CGO).

For a detailed architecture overview, see [`docs/prover-coordinator-overview.md`](docs/prover-coordinator-overview.md).

## When You Are Working On an OpenVM / zkvm-prover Upgrade

Follow the structured testing guide in [`docs/testing/openvm-upgrade-testing-guide.md`](docs/testing/openvm-upgrade-testing-guide.md). It covers five verification levels:

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

## Troubleshooting Common E2E Test Issues

### Port Conflicts (Shared Servers)
- System PostgreSQL often occupies port 5432. If the default `DB_PORT=5432` conflicts with a system instance, edit `.env` to use an alternative (e.g., `5433`) and run `make gen-config` to regenerate all configs.
- Kill stale coordinator processes before restarting: `pkill -f coordinator_api`.

### Stale Docker Containers
- After changing `docker-compose.yml`, old containers may persist with stale port mappings. Always use `docker rm -f <name>` before `docker compose up`.
- The E2E container is named `local_postgres`. Verify the port mapping with `docker port local_postgres`.

### Solc Version
- The project requires **solc ≥ 0.8.24** (for `--evm-version cancun`). System-installed solc is often older.
- Workaround: download `solc-static-linux` v0.8.24 to `/tmp/solc` and prepend `/tmp` to PATH.

### goose Migration Tool
- The E2E `setup_db` step requires `goose`. Install with: `go install github.com/pressly/goose/v3/cmd/goose@latest`.
- Ensure `$GOPATH/bin` (typically `~/go/bin`) is in PATH.

### Config Template Placeholders
- Some config templates contain literal placeholder strings (e.g., `"<serach a public rpc endpoint like alchemy>"`). Always verify the `l2geth.endpoint` field points to a reachable RPC before launching the coordinator.
- A bad endpoint causes the coordinator to panic at startup during `InitL2geth`.

### validium_mode Consistency
- The E2E config (`tests/prover-e2e/*/config.json`) and coordinator config (`coordinator/build/bin/conf/config.json`) must agree on `validium_mode`. Mismatch causes "invalid data length for DABatchV7" errors.
- For mainnet testing: set `validium_mode: false`.
- For cloak / validium testing: set `validium_mode: true` and ensure `sequencer.decryption_key` is provided.

### Fork & Block Range Selection
- Blocks must be post-fork to match the configured codec version. For GalileoV2 (codec V10) on mainnet, use blocks ≥ 33,750,000. Older blocks (e.g., 26,653,680) are Galileo (codec V9) and will fail with "mismatched post-state root".
- To verify fork compatibility: check `codec_version` in the E2E config and ensure `SCROLL_FORK_NAME` matches the coordinator's verifier fork list.

### S3 Asset URLs
- The prover config `base_url` must match the actual S3 object path. Verify with `curl -sI` before running.
- The coordinator downloads **verifier** assets from `v0.X.X/verifier/`; the prover downloads **circuit** assets from `<fork>/<proof_type>/<vk>/`.
- If you see HTTP 403 from S3, check whether the URL contains a `releases/` segment that shouldn't be there.

### Multiple Coordinator Instances
- Running `make coordinator_setup` rebuilds the binary but does not stop running instances. If the old instance holds port 8390, the new one fails with `bind: address already in use`.
- Always check with `ss -tlnp | grep 8390` before launching.

## Coordination with Humans

- **Code / logic issues**: agents should reason independently and propose fixes.
- **Environment / secrets issues** (database passwords, RPC endpoints, cloud credentials, sudo access): ask the human and wait for a response. Do not time out and make unilateral decisions.

## Documentation Index

| Document | What It Covers |
|----------|----------------|
| [`docs/prover-coordinator-overview.md`](docs/prover-coordinator-overview.md) | Architecture, data flow, component relationships, common operations |
| [`docs/testing/openvm-upgrade-testing-guide.md`](docs/testing/openvm-upgrade-testing-guide.md) | Step-by-step testing checklist after OpenVM / zkvm-prover upgrades |
| [`docs/testing/docker-compose-e2e-guide.md`](docs/testing/docker-compose-e2e-guide.md) | Production-like E2E testing with Docker Compose + Coordinator Proxy |
| [`docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md`](docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md) | Test report for PR #1783 (OpenVM 1.6.0, guest v0.8.0) |
