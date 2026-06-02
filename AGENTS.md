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

## Shadow Coordinator + Prover Testing (Production Task Replay)

For testing proof generation against **real mainnet production tasks** without interfering with the live system, use the **Shadow Coordinator** approach. This is significantly faster than a full shadow fork:

- **Architecture**: Local coordinator (`:8390`) + local prover (GPU), fed by imported production task data.
- **Docs**: [`docs/shadow-testing/README.md`](docs/shadow-testing/README.md) — full setup guide, troubleshooting, config reference.
- **Quick Start**: [`scripts/shadow-testing/QUICKSTART.md`](scripts/shadow-testing/QUICKSTART.md)
- **Automation**: [`scripts/shadow-testing/setup.sh`](scripts/shadow-testing/setup.sh) — one-command setup for postgres, coordinator, and prover.

Key hard-won rules:
- **L2 RPC for coordinator task generation** (must support `debug_executionWitness`):
  - ✅ **Primary**: `https://l2geth-rpc-proxy.mainnet.aws.scroll.io/` (internal/debug-enabled, supports `debug_executionWitness`)
  - ⚠️ **Fallback**: `https://mainnet-rpc.scroll.io` (public RPC, may not support `debug_executionWitness` for chunk task generation)
  - ❌ **Avoid**: `https://rpc.scroll.io` (does not work)
- **Alchemy API for Anvil fork** (must use Alchemy, others hit rate limits):
  - ✅ **Primary**: `https://eth-mainnet.g.alchemy.com/v2/YOUR_ALCHEMY_API_KEY`
  - 📋 **Credential source**: Check `local-secrets.md`, `.env`, or `.pgpass` first. If not found, **ask a human** — do not guess or invent keys.
- **S3 circuit URLs**: v0.8.0 uses `v0.8.0/` prefix (no `/releases/`).
- **l2_block table**: Coordinator needs this for block hash lookups. Must be populated and linked via `chunk_hash`.
- **Blocks**: Must be post-fork (GalileoV2 / codec V10 = blocks ≥ 33,750,000 on mainnet).
- **L1 messages**: If chunks contain L1 messages, prover needs `scroll_getL1MessagesInBlock` RPC support. Most chunks at current mainnet height do NOT contain L1 messages, so this is usually non-blocking.
- **Anvil MUST fork Ethereum L1, NOT Scroll L2**: The ScrollChain proxy address `0xa13BAF47339d63B743e7Da8741db5456DAc1E556` is on **Ethereum mainnet** (chainId=1), not Scroll mainnet (chainId=534352). If you accidentally point Anvil at a Scroll L2 RPC (e.g., `scroll-mainnet.g.alchemy.com`), the proxy address will have no code or wrong code, and all contract interactions will fail. Always verify `eth_chainId` returns `1` after forking.

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

## Secrets & Credentials Reference

**All sensitive endpoints, keys, and passwords for local development are documented in [`local-secrets.md`](local-secrets.md)** (git-ignored).

| Category | What's Inside | Why It Matters |
|----------|---------------|----------------|
| **RPC Endpoints** | ETH L1 (Alchemy mainnet/sepolia), Scroll L2 (public/internal) | Anvil must fork **ETH L1**, not Scroll L2. Coordinator needs debug-enabled L2 RPC. |
| **Database DSNs** | Local shadow DB (port 5433), Sepolia shadow DB (port 5442), Mainnet RDS (port 15432 via tunnel) | Wrong DSN = wrong chain data = wasted proving hours. |
| **Contract Addresses** | ScrollChain proxy, L1MessageQueueV2, RollupVerifier, MockVerifier | These change per network (mainnet vs sepolia). Hard-coding without checking = `ErrorIncorrectBatchHash`. |
| **Sender Keys** | Commit/finalize EOA private keys for shadow fork | Anvil-funded accounts; never use production keys in shadow tests. |
| **S3 URLs** | Circuit asset base URLs | v0.8.0 drops the `/releases/` prefix. Wrong URL = 403. |

> **Agent Rule**: Before starting any shadow fork or E2E test, always cross-reference `local-secrets.md`. If a required secret is missing, ask the human — do not invent URLs or credentials.

## Documentation Index

| Document | What It Covers |
|----------|----------------|
| [`docs/prover-coordinator-overview.md`](docs/prover-coordinator-overview.md) | Architecture, data flow, component relationships, common operations |
| [`docs/testing/openvm-upgrade-testing-guide.md`](docs/testing/openvm-upgrade-testing-guide.md) | Step-by-step testing checklist after OpenVM / zkvm-prover upgrades |
| [`docs/testing/docker-compose-e2e-guide.md`](docs/testing/docker-compose-e2e-guide.md) | Production-like E2E testing with Docker Compose + Coordinator Proxy |
| [`docs/shadow-testing/README.md`](docs/shadow-testing/README.md) | Shadow coordinator + local prover setup for production task replay |
| [`docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md`](docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md) | Test report for PR #1783 (OpenVM 1.6.0, guest v0.8.0) |
