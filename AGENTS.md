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
- **Docs**: [`tests/shadow-testing/docs/README.md`](tests/shadow-testing/docs/README.md) — full setup guide, troubleshooting, config reference.
- **Quick Start**: [`tests/shadow-testing/docs/QUICKSTART.md`](tests/shadow-testing/docs/QUICKSTART.md)
- **Automation**: [`tests/shadow-testing/scripts/setup.sh`](tests/shadow-testing/scripts/setup.sh) — one-command setup for postgres, coordinator, and prover.

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

### Sepolia Shadow Fork — Additional Rules

| Dimension | Mainnet | Sepolia | Trap |
|-----------|---------|---------|------|
| **DB port** | `localhost:5433` (shadow) / `15432` (RDS tunnel) | `localhost:25432` (RDS tunnel) | Wrong port = connecting to mainnet data |
| **L2 RPC** | `l2geth-rpc-proxy.mainnet.aws.scroll.io` | `l2geth-rpc-proxy.sepolia.aws.scroll.io` | Public Sepolia RPC (`sepolia-rpc.scroll.io`) rejects `debug_executionWitness` |
| **Verifier** | Mainnet has `latestVerifier[10] = 0x0dE1...` (can `anvil_setCode`) | Production proofs + production MVRV may already match | Re-using old proofs → check MVRV first. Testing **new guest** → MUST deploy fresh verifier |
| **`committedBatches`** | Sparse, but fork block usually covers target batches | Sparse; **every bundle end batch must exist** | Missing entry → `ErrorIncorrectBatchHash(0x2a1c1442)` |
| **`L1MessageQueueV2`** | Reset `nextUnfinalizedQueueIndex = 0` sufficient | Set to `MIN(total_l1_messages_popped_before)` of first target batch; **slot 104** (verify with `forge inspect`) | Wrong slot/value → `ErrorFinalizedIndexTooLarge(0x16465978)` |
| **Anvil gas estimation** | Same as mainnet | `eth_estimateGas` fails with fee caps present (`Gas=0`) | Patch `estimategas.go` or use `--min-codec-version` workaround |
| **Sender balance** | Persisted across restarts | **Resets to 0** after Anvil restart | Must re-fund EOAs before each relayer start |
| **Relayer flags** | Standard | Requires `--config <path>` AND `--min-codec-version 10` | Missing flags = wrong config or immediate exit |
| **DB scope** | Imported limited range | Full production snapshot (batches 128080+) | Relayer batch committer floods logs with commit retries |
| **Blob version** | Usually V0 | Anvil 1.0.0 cannot decode BlobSidecar V1 | Set `fusaka_timestamp: 2000000000` in relayer config |
| **Proofs in DB** | May already be v0.8.0 | Old proofs are v0.7.3 | Must reset `proving_status = 1` to regenerate with v0.8.0 |

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

## Troubleshooting: Verifier Wrapper Deployment on Shadow Forks

### `anvil_setCode` Does NOT Reset Immutables
- **Problem**: Copying a mainnet verifier wrapper (e.g., `ZkEvmVerifierPostFeynman`) to Anvil via `anvil_setCode` preserves the **original immutables** (`verifierDigest1`, `verifierDigest2`, `protocolVersion`). These digests are bound to the mainnet plonk verifier VK and will **never** match locally-generated proofs.
- **Symptom**: `VerificationFailed` (selector `0x439cc0cd`) even when the plonk verifier binary, public input hash, and proof are all individually correct.
- **Root cause**: The wrapper assembles its own `instances` array from immutables + `keccak256(protocolVersion || publicInput)`. Wrong immutables = wrong instances = plonk verifier rejects the proof.
- **Solution**: **Always recompile and redeploy** the wrapper with immutables extracted from the *local* proof's `instances` array (bytes 384–416 and 416–448 for digest1/digest2).

### Do Not "Fix" Production Solidity Without Evidence
- **Problem**: When `VerificationFailed` appears, it's tempting to blame the assembly loop in the wrapper (`sub(0x5a0, i)` vs `add(0x1c0, i)`).
- **Reality**: The production wrapper (`ZkEvmVerifierPostFeynman.sol`) has used `sub(0x5a0, i)` since deployment and has finalized thousands of bundles on mainnet. The loop direction maps hash bytes in **reverse order** to instance words, which matches the verifier circuit's expectation.
- **Symptom of wrong patch**: Changing the loop to `add(0x1c0, i)` inverts the hash-word layout, producing a different set of instances that also fail verification.
- **Correct diagnosis flow**:
  1. Verify the plonk verifier binary matches the deployed contract runtime code.
  2. Verify `keccak256(abi.encodePacked(protocolVersion, publicInput))` matches the proof metadata `bundle_pi_hash`.
  3. Verify the wrapper's immutables match the local proof's digest words.
  4. Only after (1–3) pass should you look at Solidity logic — and even then, production code is almost certainly correct.

### Access Control on `finalizeBundlePostEuclidV2`
- `ScrollChain.finalizeBundlePostEuclidV2` has `OnlyProver` modifier.
- On shadow fork, impersonate the registered prover EOA before sending the transaction: `cast rpc anvil_impersonateAccount <prover_address>`.

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

## Agent Discipline: Research Before Experimentation

> **Rule**: When encountering a problem that is **non-trivial**, **time-consuming**, or **has failed more than once**, the agent **must** search existing documentation before attempting new fixes.
>
> 1. Read all relevant markdown files in the task directory (e.g., `tests/shadow-testing/docs/*.md`, `LESSONS_LEARNED.md`).
> 2. Search for similar error messages, selectors, or symptoms in the codebase and docs.
> 3. Only after confirming the issue is **not documented** should you design a new experiment.
>
> **Why**: This repository has extensive documentation of past pitfalls. Blind experimentation wastes time and repeats mistakes that are already solved in writing.

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
| [`tests/shadow-testing/docs/README.md`](tests/shadow-testing/docs/README.md) | Shadow coordinator + local prover setup for production task replay |
| [`tests/shadow-testing/docs/LESSONS_LEARNED.md`](tests/shadow-testing/docs/LESSONS_LEARNED.md) | Hard-won debugging knowledge from past shadow tests (read before experimenting) |
| [`tests/shadow-testing/docs/QUICKSTART.md`](tests/shadow-testing/docs/QUICKSTART.md) | Quick reference for common shadow testing commands |
| [`docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md`](docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md) | Test report for PR #1783 (OpenVM 1.6.0, guest v0.8.0) |
