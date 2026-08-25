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

For testing proof generation against **real mainnet production tasks** without interfering with the live system, use the **Shadow Coordinator** approach. This is significantly faster than a full shadow fork. There are two test modes, each in its own directory under `tests/shadow-testing/`:

- **Follow mode (primary/default)** — `cd tests/shadow-testing/follow && make follow` forks the ETH mainnet state `FOLLOW_FORK_HOURS_BACK` hours in the past (default 5h; `0` = current tip), catches up the resulting backlog, then follows mainnet bundle production in real time (poll-sync → prove → finalize, default 48h window). This is the default acceptance test for prover/guest upgrades. Operations: `make follow-status`, `make follow-report`, `make follow-stop`, `make re-fork`.
- **Snapshot replay mode (specialized)** — `cd tests/shadow-testing/snapshot`: fork a historical block, import a fixed bundle range, prove & finalize ~N bundles (`make all` / `make docker-all` / `make sepolia-all`). Use for incident reproduction, single-bundle debugging, Sepolia testing, targeted codec-migration checks.

Shared details:

- **Architecture**: Local coordinator (`:8390`) + local prover (GPU), fed by imported production task data. Scripts shared by both modes live in `tests/shadow-testing/lib/`; runtime state is shared at `tests/shadow-testing/.work/`.
- **Docs**: [`tests/shadow-testing/follow/GUIDE.md`](tests/shadow-testing/follow/GUIDE.md) and [`tests/shadow-testing/snapshot/GUIDE.md`](tests/shadow-testing/snapshot/GUIDE.md) — per-mode setup guides, troubleshooting, config reference.
- **Quick Start**: [`tests/shadow-testing/README.md`](tests/shadow-testing/README.md) (mode chooser)
- **Automation**: `tests/shadow-testing/{follow,snapshot}/Makefile` — per-mode Makefile targets; the root `tests/shadow-testing/Makefile` is a thin dispatcher (`make follow-up`, `make snapshot-all ...`).

Key hard-won rules:
- **L2 RPC for coordinator task generation** (must support `debug_executionWitness`):
  - ✅ **Primary**: `https://l2geth-rpc-proxy.mainnet.aws.scroll.io/` (internal/debug-enabled, supports `debug_executionWitness`)
  - ⚠️ **Fallback**: `https://mainnet-rpc.scroll.io` (public RPC, may not support `debug_executionWitness` for chunk task generation)
  - ❌ **Avoid**: `https://rpc.scroll.io` (does not work)
- **Alchemy API for Anvil fork** (must use Alchemy, others hit rate limits):
  - ✅ **Primary**: `https://eth-mainnet.g.alchemy.com/v2/YOUR_ALCHEMY_API_KEY`
  - 📋 **Credential source**: Check `local-secrets.md`, `.env`, or `.pgpass` first. If not found, **ask a human** — do not guess or invent keys.
- **S3 circuit URLs**: v0.9.0 uses `releases/v0.9.0/` prefix. (v0.8.0 historically used `v0.8.0/` without `/releases/`.)
- **S3 digest encoding**: v0.8.0 `digest_*.hex` files are **Montgomery form** — convert to canonical before deploying a verifier; v0.9.0+ files are canonical and usable directly. See [`tests/shadow-testing/docs/bundle-digest-encoding.md`](tests/shadow-testing/docs/bundle-digest-encoding.md).
- **l2_block table**: Coordinator needs this for block hash lookups. Must be populated and linked via `chunk_hash`.
- **Blocks**: Must be post-fork (GalileoV2 / codec V10 = blocks ≥ 33,750,000 on mainnet).
- **L1 messages**: If chunks contain L1 messages, prover needs `scroll_getL1MessagesInBlock` RPC support. Most chunks at current mainnet height do NOT contain L1 messages, so this is usually non-blocking.
- **Anvil MUST fork Ethereum L1, NOT Scroll L2**: The ScrollChain proxy address `0xa13BAF47339d63B743e7Da8741db5456DAc1E556` is on **Ethereum mainnet** (chainId=1), not Scroll mainnet (chainId=534352). If you accidentally point Anvil at a Scroll L2 RPC (e.g., `scroll-mainnet.g.alchemy.com`), the proxy address will have no code or wrong code, and all contract interactions will fail. Always verify `eth_chainId` returns `1` after forking.

### Follow Mode — Mid-Run Upgrade Test

The **mid-run upgrade test** (`cd tests/shadow-testing/follow && make follow-old` … `make follow-upgrade`, script `follow/scripts/20-upgrade.sh`) simulates a production zk-stack upgrade with **continuous finalization**: Phase 1 runs the current production release against the fork's production MVRV verifier (`--skip-verifier` — no wrapper deployed), then one command swaps coordinator assets, restarts provers with the new circuits, and registers the new `ZkEvmVerifierPostFeynman` wrapper via the **genuine `updateVerifier` path** (old wrapper moves to `legacyVerifiers`) — same fork, same DB, relayer/daemons untouched. Bundles proven pre-boundary finalize through the old wrapper (real legacy-routing coverage); everything at/after the boundary batch `N` is reset and re-proven by the new stack. See `tests/shadow-testing/follow/GUIDE.md` "Mid-Run Upgrade Test" for procedure, acceptance criteria, and traps (in-flight old proofs always fail post-cutover — the reset is mandatory; the script never rebuilds binaries).

> ⚠️ **Phase 1 must run the ACTUAL production zk stack** (usually a `develop` build in a separate `git worktree`, aimed via `COORD_DIR`/`PROVER_BIN`/`ASSETS_DIR`) — never assume the current checkout == production. Verify the production guest version first (follow/GUIDE.md "Determining the Production zk Stack"; digest encoding in `tests/shadow-testing/docs/bundle-digest-encoding.md`), or Phase-1 finalizations will all fail `VerificationFailed` (Trap 31).

### Follow Mode — Canary Parallel-Upgrade Test

The **canary parallel-upgrade test** (`make follow-canary` … `make canary-upgrade`, optionally `make canary-rollback`) is an alternative to the hard-switch upgrade test: Phase A imports **production bundle proofs from the mainnet DB** (poll-sync `--import-proofs` / `SYNC_PROOFS=1`, quarantined in the shadow-private `remote_bundle_proof` table) and finalizes them through the forked production wrapper with **no local proving and no old-stack build**; `30-canary-upgrade.sh` then registers the new wrapper at boundary `N` (oldest unfinalized bundle without a remote proof) and starts the new coordinator+provers, so old (< N, imported proofs) and new (≥ N, local proofs) bundles genuinely finalize in parallel. `31-canary-rollback.sh` restores the old wrapper at N and applies the quarantined proofs. See `tests/shadow-testing/follow/GUIDE.md` "Canary Parallel-Upgrade Test" and TROUBLESHOOTING Trap 39.

### Follow Mode — Additional Rules

For continuously proving mainnet bundles in real time (poll-syncing the mainnet DB into the shadow DB), three silent starvation traps apply — see `tests/shadow-testing/follow/TROUBLESHOOTING.md` Trap 22/23:
- Poll sync must also maintain `l2_block.chunk_hash` links (UPDATE, not INSERT) and `chunk.batch_hash`/`batch.bundle_hash` parent links, or tasks become invisible to the coordinator with no ERROR logged.
- `sweep-stale-proving.sh` must reset `total_attempts`, not just `proving_status` — the coordinator skips tasks with `total_attempts >= 5`.
- Bundles popping L1 messages enqueued after the Anvil fork block fail with `VerificationFailed(0x439cc0cd)`: the queue-hash sync (`tests/shadow-testing/lib/sync-queue-hashes.py`) runs inside the `sync-mainnet-db.py` poll loop every cycle to mirror `messageRollingHashes` + `nextCrossDomainMessageIndex` from mainnet onto the fork.
- Never run a poll sync against an empty/reset DB: watermark=0 makes it backfill all of mainnet history (Trap 25). Baseline first (handled by `10-follow-up.sh`), and stop daemons with `11-follow-stop.sh` which kills whole process groups — an orphaned python sync is invisible to pidfile checks.

Daemons: poll-sync (with integrated queue-hash sync), the sweeper, and the hourly monitor all run as plain background processes with pidfiles in `.work/`, started by `follow/scripts/10-follow-up.sh` (the sweeper loops every 10 min) — **no agent/session cron is required**. Fork recovery: `make re-fork` (re-fork, redeploy wrapper, re-fund EOAs, re-mirror queue hashes, restart relayer).

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
| **Proofs in DB** | May already be v0.9.0 | Old proofs are v0.7.3 | Must reset `proving_status = 1` to regenerate with v0.9.0 |

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

# Build prover (GPU + halo2-gpu SNARK acceleration, 24 GB-class GPUs)
cd zkvm-prover && make prover_halo2gpu

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
- For v0.9.0, both coordinator **verifier** assets and prover **circuit** assets are under `scroll-zkvm/releases/v0.9.0/`. Earlier v0.8.0 assets used `scroll-zkvm/v0.X.X/` for verifier assets and `scroll-zkvm/galileov2/` for prover circuits.
- If you see HTTP 403 from S3, check whether the URL uses the correct `releases/` prefix for the target version.
- Prover circuit assets use a **flat** layout: `<base>/<circuit>/app.vmexe` (no VK subdirectory). zkvm master ≥ bf887150 additionally requires `agg_vk.bin` in each circuit dir (else the halo2-gpu prover wastes GPU memory deriving the VK), and the coordinator reads the batch circuit's `agg_vk.bin` from its assets dir — both are new build-guest artifacts that must be uploaded to S3 after a guest rebuild (see `tests/shadow-testing/follow/TROUBLESHOOTING.md` Trap 33).
- After bumping the `scroll-zkvm-*` Cargo pin, always `git diff Cargo.lock` and revert revm-family drift before building (Trap 32).

### Multiple Coordinator Instances
- Running `make coordinator_setup` rebuilds the binary but does not stop running instances. If the old instance holds port 8390, the new one fails with `bind: address already in use`.
- Always check with `ss -tlnp | grep 8390` before launching.

### Querying the Remote Mainnet DB (RDS via tunnel)
Rules learned the hard way querying the production read-only DB (`192.168.1.108:15432`, an RDS tunnel — sizes as of 2026-08):

- **Every `l2_block` index is partial: `WHERE deleted_at IS NULL`.** Any query on `l2_block` (and `chunk`) without that predicate **cannot use any index** and seq-scans the whole heap (62 GB for `l2_block`) — even innocent-looking ones like `SELECT MAX(number) FROM l2_block`. Always write `... WHERE deleted_at IS NULL AND ...`. (Same pattern on `chunk`; its indexes are partial too.)
- **Never run unbounded `count(*)` on `l2_block`.** Plain `count(*)` = parallel seq scan ≈ **2 minutes** (33.5M rows / 62 GB heap, 183 GB incl. TOAST). Even `count(*) WHERE deleted_at IS NULL` (parallel index-only scan) ran **>5 minutes** — the index itself is multi-GB. Just don't count this table.
- For row-count estimates use the planner stats instead — instant:
  ```sql
  SELECT reltuples::bigint FROM pg_class WHERE relname = 'l2_block';
  ```
- Bounded counts are fine when they carry the partial-index predicate plus a range on the indexed column: `SELECT count(*) FROM l2_block WHERE deleted_at IS NULL AND number > X AND number <= Y;` (an 8.4k-block range ≈ 8 s). `MAX(number) ... WHERE deleted_at IS NULL` is instant.
- `chunk` / `batch` / `bundle` are small enough (heaps ≤ 5 GB) that `count(*) WHERE deleted_at IS NULL` returns in seconds — but still always include the predicate, or you seq-scan.
- Each `psql` invocation pays ~0.7 s TLS+SCRAM setup and ~80 ms RTT per query (the tunnel forwards to a remote RDS; the LAN hop itself is <1 ms). Batch multiple statements into one session (one `psql` with several `-c`, or interactive) instead of one invocation per query.
- The follow-mode poll-sync daemon only issues indexed queries (`MAX(index)`, range-bound `COPY`, small proof windows), so it is unaffected — this section is for ad-hoc/manual queries.

## Agent Discipline: Research Before Experimentation

> **Rule**: When encountering a problem that is **non-trivial**, **time-consuming**, or **has failed more than once**, the agent **must** search existing documentation before attempting new fixes.
>
> 1. Read all relevant markdown files in the task directory (e.g., `tests/shadow-testing/docs/*.md`).
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
| **S3 URLs** | Circuit asset base URLs | v0.9.0 uses the `releases/v0.9.0/` prefix. Wrong URL = 403. |

> **Agent Rule**: Before starting any shadow fork or E2E test, always cross-reference `local-secrets.md`. If a required secret is missing, ask the human — do not invent URLs or credentials.

## Documentation Index

| Document | What It Covers |
|----------|----------------|
| [`docs/prover-coordinator-overview.md`](docs/prover-coordinator-overview.md) | Architecture, data flow, component relationships, common operations |
| [`docs/testing/openvm-upgrade-testing-guide.md`](docs/testing/openvm-upgrade-testing-guide.md) | Step-by-step testing checklist after OpenVM / zkvm-prover upgrades |
| [`docs/testing/docker-compose-e2e-guide.md`](docs/testing/docker-compose-e2e-guide.md) | Production-like E2E testing with Docker Compose + Coordinator Proxy |
| [`tests/shadow-testing/follow/GUIDE.md`](tests/shadow-testing/follow/GUIDE.md) | Follow mode: shadow coordinator + local prover following live mainnet (primary acceptance test) |
| [`tests/shadow-testing/snapshot/GUIDE.md`](tests/shadow-testing/snapshot/GUIDE.md) | Snapshot replay mode: historical fork + fixed bundle range (incident reproduction, Sepolia) |
| [`tests/shadow-testing/docs/COMMON-TROUBLESHOOTING.md`](tests/shadow-testing/docs/COMMON-TROUBLESHOOTING.md) | Mode-independent pitfalls and agent checklists for shadow testing; per-mode traps live in `tests/shadow-testing/follow/TROUBLESHOOTING.md` and `tests/shadow-testing/snapshot/TROUBLESHOOTING.md` |
| [`tests/shadow-testing/README.md`](tests/shadow-testing/README.md) | Quick reference for common shadow testing commands |
| [`docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md`](docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md) | Test report for PR #1783 (OpenVM 1.6.0, guest v0.8.0) |
| [`docs/testing/single-chunk-reproving.md`](docs/testing/single-chunk-reproving.md) | Re-proving one specific mainnet chunk using shadow DB + local coordinator/prover |
