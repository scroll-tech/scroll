# Agent Instructions for Scroll Monorepo

## Quick Orientation

Mixed **Rust + Go** monorepo for the Scroll ZK Rollup. Key components for proving work:

- **Coordinator** (`coordinator/`) — Go service scheduling proving tasks and verifying proofs
- **Prover** (`crates/prover-bin/`) — Rust binary generating ZK proofs via OpenVM
- **Shared library** (`crates/libzkp/`, `crates/libzkp_c/`) — proof logic consumed by both (CGO)

Architecture overview: [`docs/prover-coordinator-overview.md`](docs/prover-coordinator-overview.md).

## Testing Workflows (pick the right one)

| Workflow | When | Entry point |
|---|---|---|
| Upgrade testing ladder (build → unit → artifact → E2E → docker) | after OpenVM / zkvm-prover upgrades | [`docs/testing/openvm-upgrade-testing-guide.md`](docs/testing/openvm-upgrade-testing-guide.md) |
| **Shadow follow mode** — fork mainnet, follow live bundle production (default acceptance test) | prover/guest changes | [`tests/shadow-testing/follow/GUIDE.md`](tests/shadow-testing/follow/GUIDE.md) |
| **Shadow canary parallel-upgrade** — old/new stacks finalize side by side, with rollback drill | pre-upgrade gate (cheap, no old-stack build) | follow/GUIDE.md "Canary Parallel-Upgrade Test" |
| **Shadow hard-switch upgrade** — production-cutover rehearsal (in-flight task reset) | mandatory for major upgrades | follow/GUIDE.md "Mid-Run Upgrade Test" |
| **Shadow snapshot replay** — fixed historical bundle range | incident reproduction, single-bundle debug, Sepolia | [`tests/shadow-testing/snapshot/GUIDE.md`](tests/shadow-testing/snapshot/GUIDE.md) |
| Local E2E (no fork) | quick pipeline sanity | [`tests/prover-e2e/README.md`](tests/prover-e2e/README.md) |

Mode chooser + commands: [`tests/shadow-testing/README.md`](tests/shadow-testing/README.md).

## Non-Negotiable Rules (details behind each link)

- **Anvil forks Ethereum L1, never Scroll L2** (ScrollChain proxy `0xa13B…E556` is on ETH mainnet; verify `eth_chainId == 1`). — COMMON Trap 2
- **L2 RPC must support `debug_executionWitness`** (internal proxies only; `rpc.scroll.io` does not work). — COMMON Trap 3
- **Version-sensitive facts** (S3 prefixes, digest encoding, codec block thresholds, production stack identity) live in ONE place: [`tests/shadow-testing/docs/CURRENT-STACK.md`](tests/shadow-testing/docs/CURRENT-STACK.md). Never hard-code them elsewhere.
- **Verify the production zk stack before upgrade tests** — never assume the current checkout == production (follow/GUIDE "Determining the Production zk Stack"). — Trap 31
- **Querying the production RDS**: partial-index and `count(*)` rules in [`tests/shadow-testing/docs/rds-query-rules.md`](tests/shadow-testing/docs/rds-query-rules.md) — violations cost real money.
- **Stop test stacks when done** (`make follow-stop`) — an idle stack polls RDS forever.

## Useful Commands

```bash
cargo fmt --all -- --check && cargo clippy --all-features --all-targets -- -D warnings
cargo build --release -p libzkp-c            # shared lib for coordinator (CGO)
make -C zkvm-prover prover                   # GPU prover  (prover_cpu / prover_halo2gpu variants)
make -C coordinator coordinator_api coordinator_cron
make -C coordinator test                     # unit tests (needs libzkp.so)
cd tests/prover-e2e && ln -snf mainnet-galileoV2 conf && make all && make coordinator_setup
```

## Directory Guide

| Directory | Purpose |
|-----------|---------|
| `crates/libzkp`, `crates/libzkp_c` | Core proving/verification library + C FFI |
| `crates/prover-bin` | Prover binary (`prover`) |
| `coordinator/` | Go coordinator service |
| `rollup/` | Go rollup services (task production, relayer) |
| `tests/shadow-testing/` | Shadow fork/follow/canary/snapshot harness (start at its README) |
| `tests/prover-e2e/` | Local E2E harness (coordinator + prover, no fork) |
| `tests/integration-test/` | General integration tests |
| `zkvm-prover/` | Prover build scripts + runtime config |
| `build/dockerfiles/` | Production Dockerfiles |

## Agent Discipline: Research Before Experimentation

> **Rule**: When encountering a problem that is **non-trivial**, **time-consuming**, or **has failed more than once**, search existing documentation before attempting new fixes.
>
> 1. Read the relevant markdown in the task directory (e.g., `tests/shadow-testing/docs/*.md`).
> 2. Search for similar error messages, selectors, or symptoms in codebase and docs — start from the **trap index**: [`tests/shadow-testing/docs/TROUBLESHOOTING.md`](tests/shadow-testing/docs/TROUBLESHOOTING.md) (56 numbered traps with symptom tables).
> 3. Only after confirming the issue is **not documented** should you design a new experiment.
>
> **Why**: this repo documents its pitfalls extensively. Blind experimentation repeats solved mistakes. When you discover a new trap or workaround, **write it back** (see the documentation conventions in `tests/shadow-testing/README.md`).

## Coordination with Humans

- **Code / logic issues**: reason independently and propose fixes.
- **Environment / secrets issues** (DB passwords, RPC endpoints, cloud credentials, sudo): ask the human and wait. Do not time out and make unilateral decisions.

## Secrets & Credentials

All local-development secrets are in [`local-secrets.md`](local-secrets.md) (git-ignored). Before any shadow/E2E test, cross-reference it; if a secret is missing, **ask a human** — never invent URLs or credentials. Categories inside: RPC endpoints (ETH L1 / Scroll L2), DB DSNs (shadow / Sepolia / mainnet RDS tunnel), contract addresses per network, sender EOA keys, S3 base URLs.

## Documentation Index

| Document | Covers |
|----------|--------|
| [`docs/prover-coordinator-overview.md`](docs/prover-coordinator-overview.md) | Architecture, data flow, common operations |
| [`docs/testing/openvm-upgrade-testing-guide.md`](docs/testing/openvm-upgrade-testing-guide.md) | Five-level upgrade verification ladder |
| [`tests/shadow-testing/README.md`](tests/shadow-testing/README.md) | Mode chooser + documentation conventions |
| [`tests/shadow-testing/docs/TROUBLESHOOTING.md`](tests/shadow-testing/docs/TROUBLESHOOTING.md) | **Trap index** (all 56 traps → per-file locations) |
| [`tests/shadow-testing/docs/COMMON-TROUBLESHOOTING.md`](tests/shadow-testing/docs/COMMON-TROUBLESHOOTING.md) | Mode-independent traps, pre-flight ritual, checklists, symptom table |
| `tests/shadow-testing/{follow,snapshot}/GUIDE.md` + `TROUBLESHOOTING.md` | Per-mode runbooks and traps |
| [`tests/shadow-testing/docs/CURRENT-STACK.md`](tests/shadow-testing/docs/CURRENT-STACK.md) | Dated version-sensitive facts (S3, digests, thresholds, production identity) |
| [`tests/shadow-testing/docs/rds-query-rules.md`](tests/shadow-testing/docs/rds-query-rules.md) | Production RDS query discipline |
| [`docs/testing/single-chunk-reproving.md`](docs/testing/single-chunk-reproving.md) | Re-proving one mainnet chunk |
| `docs/testing_reports/*.md` | Dated test reports: [canary 2026-09-11](docs/testing_reports/canary-parallel-upgrade-2026-09-11.md) · [canary docker 2026-08-31](docs/testing_reports/canary-parallel-upgrade-docker-2026-08-31.md) · [canary 2026-08-25](docs/testing_reports/canary-parallel-upgrade-2026-08-25.md) · [snapshot early experiments](docs/testing_reports/snapshot-early-dryrun-and-relayer-tests.md) · [OpenVM 1.6.0 / guest v0.8.0](docs/testing_reports/openvm-v1.6.0-guest-v0.8.0-May19.md) |
