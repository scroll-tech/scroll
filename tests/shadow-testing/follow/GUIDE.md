# Follow Mode Guide — Shadow Coordinator + Prover Testing

This guide documents the **follow mode** of the shadow coordinator + local prover environment: fork the ETH mainnet state from `FOLLOW_FORK_HOURS_BACK` hours in the past (default 5h), catch up the resulting backlog, then follow mainnet bundle production in real time. This is the **default acceptance test for prover/guest upgrades**.

For a fixed historical bundle range (incident reproduction, single-bundle debugging, Sepolia testing, targeted codec-migration checks), see the [Snapshot Replay Mode Guide](../snapshot/GUIDE.md). Shared scripts live in [`../lib/`](../lib/), pitfalls in [`./TROUBLESHOOTING.md`](./TROUBLESHOOTING.md) (follow mode) and [`../docs/COMMON-TROUBLESHOOTING.md`](../docs/COMMON-TROUBLESHOOTING.md) (mode-independent). Runtime state (`.work/`) is shared by both modes at `tests/shadow-testing/.work` (i.e. `../.work` from here).

## Architecture

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  Production RDS  │     │  Shadow DB       │     │  Shadow          │
│  (read-only via  │────▶│  (local :5433)   │────▶│  Coordinator     │
│   port-forward)  │     │                  │     │  (localhost:8390)│
└──────────────────┘     └──────────────────┘     └────────┬─────────┘
                                                           │
                                                           │ assigns tasks
                                                           ▼
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  L2 RPC          │     │  Local Prover    │     │  Verifier Assets │
│  (mainnet-rpc.   │◀────│  (GPU/CPU)       │     │  (/tmp/shadow-   │
│   scroll.io)     │     │                  │     │   verifier-assets)│
└──────────────────┘     └──────────────────┘     └──────────────────┘
```

## Prerequisites

### Hardware
- GPU with CUDA support (tested on RTX 3090)
- ~50GB disk space for Docker images + verifier assets + circuit downloads
- 16GB+ RAM

### Software
- Docker + docker-compose
- PostgreSQL client (`psql`)
- Rust toolchain (for local prover binary)
- `kubectl` or SSH access to IDC for port-forwarding to production RDS

### Network
- Access to IDC machine with port-forward to mainnet RDS (e.g., `idc-us-1-19`)
- Internet access for L2 RPC and S3 circuit downloads

# Follow Mode (Primary)

Follow mode forks the ETH mainnet state from **`FOLLOW_FORK_HOURS_BACK` hours in the past** (default 5; `0` = fork at the current tip) with Anvil, baseline-syncs the shadow DB from the mainnet read replica, then **catches up the backlog** (every bundle committed-but-not-finalized at the fork block, plus everything mainnet produced since) before **following** mainnet indefinitely — poll-syncing new chunk/batch/bundle rows, proving them with local GPU provers, and finalizing bundles on the fork — until a preset duration (`FOLLOW_RUN_HOURS`, default 48) or a failure. It answers the question a snapshot cannot: *can this prover fleet catch up a backlog and then keep pace with real mainnet bundle production?* The fork block is `tip - FOLLOW_FORK_HOURS_BACK*300` (12s L1 blocks); `lastFinalizedBatchIndex` and `lastCommittedBatchIndex` are read **at the fork block**, so the baseline window (fork-finalized → mainnet tip) is exactly the backlog. `make follow-report` splits metrics into catch-up and steady-state phases using `MAINNET_BUNDLE_TIP_AT_START` recorded in `.work/follow-run.env`.

For a fixed historical bundle range (incident reproduction, single-bundle debugging, Sepolia, codec-migration checks), use the [Snapshot Replay Mode Guide](../snapshot/GUIDE.md).

## Follow Mode Prerequisites

All general prerequisites above apply, plus:

- **Mainnet RDS read-replica access** — the `~/.pgpass` file on this machine contains valid credentials for the mainnet RDS read-only replica. Verify:
  ```bash
  psql -h localhost -p 15432 -U mainnet_infra_team_read_only -d mainnet_rollup -c "SELECT COUNT(*) FROM batch;"
  # → 517,830+ batches
  ```
- **RDS tunnel watchdog** — the IDC port-forward to RDS (`localhost:15432`) must stay up for the entire run; a dead tunnel silently starves poll-sync. Run it under `autossh` (e.g. `autossh -M 0 -N -L 15432:...`) so it self-heals.
- Enough shadow-DB disk headroom for continuous row growth.

For automated one-off baseline copies, `scroll-devnets/charts/shadow-fork/rollup-relayer/scripts/copy-db.sh` streams data from mainnet RDS to the local shadow DB via `postgres-tunnel` (`COPY ... TO STDOUT | COPY ... FROM STDIN`).

## Bring-Up: `make follow`

One command brings up the entire follow-mode stack (`scripts/10-follow-up.sh`):

1. **Baseline sync** — `sync-mainnet-db.py` baseline mode copies ONLY the finalization-lag window: batches from the fork's `lastFinalizedBatchIndex - 1` to the mainnet tip, plus their chunks/`l2_block`/`l1_message` and overlapping bundles (typically ~100 batches / a few thousand blocks — NOT deep history). This window is not optional: L1 finalization is sequential (`prevBatchHash` chaining), so the shadow must prove and finalize every committed-but-not-finalized bundle in order before it can follow new ones. Rows at/below the finalized boundary are marked done (`rollup_status=5`, `proving_status=4`) and rows above are aligned to the fork's committed boundary, so a reused DB never carries stale state across forks. `10-follow-up.sh --reset` truncates the task tables first (use it whenever the DB may hold leftovers from older runs/imports — otherwise the coordinator treats ancient pending batches as work). Proof columns are never copied; everything is re-proven locally.
2. **Anvil fork** — fork ETH **L1** at the current block (never Scroll L2 — Trap 2), with `fusaka_timestamp: 2000000000` in the relayer config so Anvil accepts blob sidecars.
3. **Verifier wrapper** — deploy/register the `ZkEvmVerifierPostFeynman` wrapper matching the guest under test (manual procedure: "Real Verifier Deployment" in the [Snapshot Replay Mode Guide](../snapshot/GUIDE.md#real-verifier-deployment)).
4. **Coordinator** — start `coordinator_api` + `coordinator_cron` against the shadow DB.
5. **Provers** — start one prover per GPU.
6. **Relayer** — start the rollup relayer with local proposers **disabled** (`l2_config.{chunk,batch,bundle}_proposer_config.disable = true`): the relayer then only commits/finalizes what the DB contains, instead of synthesizing bundles at an unrealistic rate. The shadow relayer config template (`lib/configs/relayer.json.template`) additionally sets `l2_config.disable_l2_watcher = true` (the poll sync, not the relayer's L2 watcher, keeps `l2_block` populated — Trap 26) and `sender_config.chain_nonce_only = true` (sender nonces initialize from the fork, never from stale `pending_transaction` rows — Trap 7/27).
7. **Background daemons** — poll-sync, sweeper, hourly monitor (plain background processes with pidfiles in `.work/`; no agent/session cron is required).

The run window is recorded in `.work/follow-run.env` (`SHADOW_REPORT_START`, `FOLLOW_RUN_HOURS=48`). Tear down with `make follow-stop` (`scripts/11-follow-stop.sh`, supports `--keep-anvil`).

### Critical Behavior of the Poll-Sync (Do Not Bypass)

- Newly inserted rows carry **mainnet's** `rollup_status`/commit/finalize columns, which are meaningless on the fork. The sync resets them per row range:
  - `batch.index <= fork miscData.lastCommittedBatchIndex` → `rollup_status = 3` (already committed on the fork).
  - `batch.index > boundary` → `rollup_status = 1` so the shadow relayer commits them on Anvil (requires `parentBatchHash == committedBatches[lastCommittedBatchIndex]`; keep Trap 19's boundary accurate).
  - `bundle` → always `rollup_status = 1`.
- The boundary is queried from Anvil each poll cycle (`ANVIL_RPC` / `SCROLL_CHAIN` env vars to override), so it advances automatically as the shadow relayer commits new batches.
- `ON CONFLICT DO NOTHING` everywhere: rows already advanced by the shadow relayer are never overwritten. **Consequence**: columns populated lazily on mainnet after the row is first copied stay stale in the shadow DB. The sync re-derives them every cycle instead:
  - `sync_l2_blocks()` — links `l2_block.chunk_hash` for recent chunks (the blocks usually exist from baseline import but with NULL `chunk_hash`; an UPDATE, not an INSERT, is what fixes it). Missing this starves chunk task formatting (Trap 22).
  - `sync_parent_links()` — re-derives `chunk.batch_hash` and `batch.bundle_hash` from parent index ranges (mainnet sets them at proposal time; rows copied before that keep NULL forever and are invisible to the coordinator's proof-status promotion, Trap 22).
- **L1 message queue follow-along (mandatory)**: bundles that pop L1 messages enqueued after the fork block fail finalization (`VerificationFailed` / `ErrorFinalizedIndexTooLarge`, Trap 23). The poll loop runs `../lib/sync-queue-hashes.py` every cycle — it copies `getMessageRollingHash(i)` from a mainnet RPC into the fork's `messageRollingHashes` mapping (slot 101) and aligns `nextCrossDomainMessageIndex` (slot 103) with mainnet.
- **Watch item**: the first bundle whose batches were committed on mainnet **after** the fork block exercises the relayer's commit path on Anvil (blob-carrying `commitBatches` tx). Keep `fusaka_timestamp: 2000000000` in the relayer config so Anvil accepts the blob sidecar.

## Day-to-Day Operations

| Command | What It Does |
|---------|--------------|
| `make follow-status` | Latest monitor snapshot + lag summary |
| `make follow-report` | Report via `generate-catchup-report.py`, windowed to this run by `SHADOW_REPORT_START` (the metrics log accumulates across runs) |
| `make follow-stop` | Tear down the stack (`--keep-anvil` keeps the fork) |
| `make re-fork` | Recover from a fork desync (see Failure Recovery below) |

### Background Daemons

| Daemon | Script | Cadence | Purpose |
|--------|--------|---------|---------|
| Poll-sync | `sync-mainnet-db.py --poll-interval 60` | 60s loop | DB row sync + `l2_block`/parent-link repair; runs `../lib/sync-queue-hashes.py` every cycle (L1 queue rolling hashes + cursor follow mainnet, Trap 23) |
| Sweeper | `scripts/sweep-stale-proving.sh` | 10 min | Reset stale proving rows **and `total_attempts`** (attempt exhaustion silently starves tasks, Trap 22; also the safety net for the Trap 24 structural gap) |
| Monitor | `scripts/monitor-catchup.py >> .work/catchup-metrics.log` | 1 h | Hourly metrics snapshot: `finalized_lag`, alerts when lag > 3, verifier `protocolVersion` drift detection |

All daemons are plain background processes with pidfiles in `.work/`, started by `10-follow-up.sh` — no agent/session cron is required.

## Expected Steady State

Mainnet produces ~1 bundle/hour; the pipeline proves + finalizes a bundle in ~10 minutes, so once the initial backlog is cleared the system idles most of the time — provers polling with `CoordinatorEmptyProofData` every ~20s is the **normal** idle state, not an error.

Validation evidence from the first 48-hour follow run (4× GPU box):

| Metric | Value |
|--------|-------|
| Duration | ~22 h of operation |
| Bundles proved **and finalized** | 90 (zero backlog at end, lag ≤ 1) |
| Chunks / batches proved | 333 / 92 |
| Avg proof time | chunk 117 s, batch 25 s, bundle 335 s |
| Mainnet cadence vs prove+finalize | ~1 bundle/hour vs ~10 min → large headroom |

Sizing conclusion: **1 GPU suffices** (~26% duty cycle), **2 recommended** for burst/backlog absorption, **4 is comfortable headroom**. The current 4-GPU box can follow Scroll mainnet in real time.

## Failure Recovery

- **`make re-fork`** (`scripts/re-fork.sh`) — when the fork desyncs (Anvil crash/restart, state drift): re-fork Anvil at the latest block, redeploy the verifier wrapper, re-fund EOAs (Trap 10 — `anvil_setBalance` does not survive restarts), re-mirror L1 queue hashes (`sync-queue-hashes.py`, Trap 23), restart the relayer.
- **RDS tunnel watchdog** — keep the RDS port-forward under `autossh`; poll-sync stalls silently if the tunnel dies.
- **Verifier-drift alerting** — the hourly monitor snapshot detects verifier `protocolVersion` drift. If it fires, re-check the deployed wrapper and MVRV routing (Snapshot guide, "[Verify MVRV Routing](../snapshot/GUIDE.md#verify-mvrv-routing)") before trusting any finalize result.

## Mid-Run Upgrade Test (Continuous Finalization Across a zk Upgrade)

This scenario answers a question a plain follow run cannot: *can the system keep finalizing bundles continuously while the zk stack (guest circuits / prover / on-chain verifier) is upgraded underneath it?* It mirrors a production upgrade: Phase 1 runs the **current production release**, then a single cutover switches to the **new release** — same Anvil fork, same DB, daemons and relayer untouched, `lastFinalizedBatchIndex` advancing the whole time.

Why it adds coverage over a fresh follow run:

- **MVRV legacy routing is genuinely exercised** — the new wrapper is registered via the real `updateVerifier(10, N, wrapper)` contract call, so batches below the boundary `N` still route to the old wrapper through `legacyVerifiers`. (A normal follow run force-overwrites the `latestVerifier` storage slot and never tests routing.)
- **Mixed finalization** — bundles already proven with the old circuits finalize *after* the upgrade event, through the old wrapper, interleaved with new-stack bundles.
- **The operational runbook itself** — coordinator assets swap + restart (with its keygen window), prover fleet rollover, in-flight task reset.

### Procedure

> ⚠️ **Phase 1 must run the ACTUAL production zk stack, not the current
> checkout.** Do not trust `mainnet.json.template`'s `s3_base_url` or your
> branch's zkvm version to tell you what production runs — verify it (see
> "Determining the Production zk Stack" below). On 2026-07 we learned this
> the hard way: the branch under test was already v0.9.0 while mainnet still
> ran guest v0.8.0, so a Phase-1 run built from the branch produced proofs
> the production wrapper could never verify (`VerificationFailed`).

**Step 0 — determine the production stack and build it.** Production is
usually `develop`; the new stack is your feature branch. Build each in its
own checkout so both binary sets coexist — a git worktree is ideal:

```bash
cd ~/scroll  # the repo with your feature branch checked out
git worktree add ../scroll-develop develop
cd ../scroll-develop
cargo build --release -p libzkp-c
make -C coordinator coordinator_api coordinator_cron
make -C zkvm-prover prover          # GPU build, ~30-60 min

# Production verifier assets (mind the S3 prefix: v0.8.0 has NO /releases/)
mkdir -p coordinator/build/bin/assets_v0.8.0
for f in verifier.bin root_verifier_vk openVmVk.json; do
  curl -fsSL "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/v0.8.0/verifier/$f" \
    -o "coordinator/build/bin/assets_v0.8.0/$f"
done
# Runtime conf: copy build/bin/conf/config.json from your main checkout but
# point verifiers[].assets_path at assets_v0.8.0, and copy conf/genesis.json.
```

Point `follow/configs/mainnet.json` at the production release:
`prover.s3_base_url = .../scroll-zkvm/v0.8.0/` (no `/releases/`),
`assets.assets_v2 = <develop-worktree>/coordinator/build/bin/assets_v0.8.0`.

**Step 1 — Phase 1 bring-up (old stack).** The forked production MVRV
already routes to a verifier matching production circuits, so NO wrapper is
deployed (`--skip-verifier` asserts routing instead). The env overrides aim
the stack at the production worktree:

```bash
cd tests/shadow-testing/follow
COORD_DIR=<develop-worktree>/coordinator/build/bin \
PROVER_BIN=<develop-worktree>/target/release/prover \
ASSETS_DIR=<develop-worktree>/coordinator/build/bin/assets_v0.8.0 \
  ./scripts/10-follow-up.sh --skip-verifier --reset
# ... let it follow mainnet until a few bundles have finalized (lag <= 1) ...
```

**Step 2 — prepare the new stack (feature branch, usually your main
checkout):**

```bash
cd ~/scroll && make -C zkvm-prover prover   # new prover binary
# or, on 24 GB-class GPUs, with GPU-accelerated SNARK (bundle) proving:
#   make -C zkvm-prover prover_halo2gpu
make -C coordinator coordinator_api coordinator_cron   # if coordinator changed
# Download the NEW verifier assets into the dir named by
# mainnet-next.json's assets.assets_v2, and fill in mainnet-next.json
# (prover.s3_base_url of the new release, e.g. .../releases/v0.9.0/).
```

New-stack checklist (learned on the v0.9.0/halo2-gpu upgrade, 2026-07):

- **Bumping the zkvm pin** (e.g. `tag = "v0.9.0"` → `rev = <master>`): after
  `cargo update -p scroll-zkvm-*`, ALWAYS `git diff Cargo.lock` and revert any
  revm-family drift before building — `cargo update` silently breaks the
  `[patch]` fork resolution (TROUBLESHOOTING.md Trap 32).
- **`halo2-gpu` cargo feature** (zkvm master ≥ bf887150): runs the SNARK
  bundle prover on GPU too. Scroll-side it is
  `prover --features halo2-gpu` (superset of `cuda`), packaged as the
  `prover_halo2gpu` Makefile target. Needs a 24 GB-class GPU.
- **New mandatory assets**: the prover downloads `agg_vk.bin` next to each
  circuit's `app.vmexe` (flat S3 layout), and the coordinator reads the batch
  circuit's `agg_vk.bin` from its own assets dir. Both are produced by
  `make build-guest` in the zkvm-prover repo and must be uploaded to S3 /
  copied into `assets_v2` before the cutover (TROUBLESHOOTING.md Trap 33).
- **Re-run `make build-guest`** in the zkvm-prover checkout after switching
  commits, and compare `digest_*.hex` against the canonical values — a stale
  build once produced a wrong `digest_1` (Trap 33).

**Step 3 — the cutover (one command).** Aim COORD_DIR at the NEW binaries;
`PROVER_BIN` defaults to `<main-checkout>/target/release/prover`:

```bash
cd tests/shadow-testing/follow
COORD_DIR=<main-checkout>/coordinator/build/bin \
  ./scripts/20-upgrade.sh --next-config configs/mainnet-next.json
# or simply: make follow-upgrade   (uses COORD_DIR from the environment)
```

### Determining the Production zk Stack

Ground truth, in increasing order of effort:

1. **On-chain wrapper vs S3 digests** — `cast call $MVRV "latestVerifier(uint256)(uint64,address)" 10 --rpc-url <mainnet>` gives the production wrapper; compare its `verifierDigest1/2()` against the candidate release's S3 `digest_*.hex`. For v0.8.0 remember the S3 files are Montgomery-encoded — see [`../docs/bundle-digest-encoding.md`](../docs/bundle-digest-encoding.md).
2. **A real production proof** — pull the newest `bundle.proof` JSON from the production DB, base64-decode `proof.instances`, read digest words at bytes 384–416/416–448 (always canonical). The `git_version` field names the guest build commit.
3. **Repo pins** — `git show develop:Cargo.lock | grep zkvm-prover` vs your branch's, plus `zkvm-prover/print_high_zkvm_version.sh` (run it from inside `zkvm-prover/`).

`20-upgrade.sh` computes the boundary batch `N` from the DB: bundles already proven (old circuit) but not yet finalized keep their proofs and stay on the old verifier; everything at/after `N` is reset (`proving_status=1`, attempts cleared) and re-proven with the new stack. If proven and unproven bundles interleave so no clean cut exists, it falls back to a hard cutover at `lastFinalized+1` (re-proving all pending bundles) and logs a warning. It then stops provers + coordinators, swaps the coordinator's `galileoV2` assets to the new release, restarts the coordinators, deploys + registers the new wrapper via the genuine `updateVerifier` path, restarts the provers with the new circuit version, and records `UPGRADE_AT_BATCH`/`UPGRADE_AT_TIME` in `.work/follow-run.env`.

### Acceptance criteria

- Every bundle with `end_batch < N` finalizes via the **old** wrapper (check `getVerifier(10, end_batch)` against `UPGRADE_OLD_WRAPPER` in `follow-run.env`).
- The first bundle with `end_batch >= N` finalizes via the **new** wrapper — the key moment.
- `finalized_lag` returns to ≤ 1 after the re-proving burst (coordinator keygen + re-proving pending tasks causes a temporary lag spike; that is expected, mirroring production deploy downtime).
- `make follow-report` splits metrics into pre/post upgrade phases.

### Notes & traps

- **In-flight old proofs always fail after the cutover.** The coordinator re-wraps submitted universal proofs with the VK it has *at submission time* (same fork name = single VK slot, `coordinator/internal/logic/submitproof/proof_receiver.go`), so proofs from old-circuit tasks are rejected once the new assets are live. This is why step (e) resets all task rows ≥ N — do not skip it.
- **Prover-local proof caches** (`.work/prover-*/db`) are wiped for the same reason (Trap 21: stale proofs replay as VData mismatches).
- **The reset in step (e) is deliberately aggressive** (Trap 35): it resets ready flags and clears proof blobs for ALL unfinalized rows at/after N — including rows that were never assigned — and deletes in-flight `prover_task` assignment rows. Anything less leaves old-format proofs reachable by the new coordinator, which then errors on every poll and (via the priority dispatcher) starves all task types.
- **First task generation after an assets swap is slow.** The coordinator fetches `debug_executionWitness` block-by-block (~550 blocks ≈ 10-15 min for two chunks) at 0% CPU — it is not hung, and the prover-side `connection_timeout_sec = 1800` covers it. Subsequent tasks reuse no cache but generate in seconds-to-minutes.
- **The prover binary is whatever is at `target/release/prover`.** The script logs its build time and git rev but does not rebuild it — a stale binary silently re-runs the old stack.
- The monitor's verifier-drift check follows MVRV routing (`getVerifier(10, lastFinalized+1)`), so it tracks whichever wrapper currently serves the next batch — no false alarm after the cutover.
- Sepolia variant: same flow with the Sepolia configs, but mind the Sepolia table in the root AGENTS.md (EOA re-funding, `--min-codec-version`, etc.).

## Canary Parallel-Upgrade Test (Old and New Stacks Finalizing Side by Side)

The canary mode answers a stronger question than the hard-switch upgrade test: *can the old and new zk stacks finalize bundles in parallel on the same chain, split purely by MVRV batch-index routing — with a clean rollback path?* It is an **extension** of follow mode; the hard-switch `20-upgrade.sh` flow above is untouched.

How it differs from the hard-switch upgrade test:

| Dimension | Hard switch (`follow-old` → `follow-upgrade`) | Canary (`follow-canary` → `canary-upgrade`) |
|-----------|-----------------------------------------------|----------------------------------------------|
| Old proofs | Produced locally by a Phase-1 production build | Imported from the mainnet DB into `remote_bundle_proof` |
| Phase 1/Phase A build | Must build the production worktree (Trap 31) | **No old-stack build needed** |
| At the boundary N | Rows ≥ N are reset and re-proven; in-flight old proofs are discarded | **Nothing is reset** — < N finalizes on imported proofs, ≥ N is proven for the first time by the new stack |
| Rollback | Not supported | `canary-rollback` restores the old wrapper at N and applies quarantined proofs — "as if the upgrade never happened" |

Semantics: poll-sync (with `SYNC_PROOFS=1`) upserts every remotely-proved bundle (`proving_status=4`) into the shadow-private quarantine table `remote_bundle_proof`, then applies proofs to local `bundle` rows with `proof IS NULL AND rollup_status <> 5 AND end_batch_index < boundary`. The boundary starts at +∞ (Phase A) and is flipped to N by `30-canary-upgrade.sh` via `.work/canary.env` (`PROOF_IMPORT_MAX_END_BATCH`) — no daemon restart needed. After t1, remote proofs ≥ N keep accumulating in the quarantine table, which is what makes rollback safe. The relayer is completely unaware of any of this (MVRV routing happens on-chain).

### Runbook

```bash
cd tests/shadow-testing/follow

# Phase A — old system only, zero local proving:
make follow-canary            # = 10-follow-up.sh --import-proofs
# ... watch the backlog finalize purely on imported production proofs ...

# t1 — the upgrade point (new stack must already be built; see Step 2 of the
# hard-switch section for the new-stack checklist):
make canary-upgrade           # = 30-canary-upgrade.sh --next-config configs/mainnet-next.json

# t1 with DOCKER provers instead of bare metal (production-style packaging):
# build the image from THIS checkout first, then pass it in:
docker build -f build/dockerfiles/prover.Dockerfile \
    -t scrolltech/prover:$(target/release/prover --version | awk '{print $NF}') .
make canary-upgrade DOCKER_PROVERS=1 PROVER_IMAGE=scrolltech/prover:<tag>

# Parallel period — old (< N) and new (>= N) bundles interleave on-chain.
# Rollback drill (recommended BEFORE the first >= N bundle finalizes):
make canary-rollback          # = 31-canary-rollback.sh
make canary-upgrade           # upgrade again afterwards — 30 is re-runnable
```

`30-canary-upgrade.sh` computes `N = MIN(end_batch_index)` over unfinalized bundles with **no** quarantined remote proof (fallback `lastFinalized+1`; override with `--start-batch`), so no local GPU time is wasted re-proving what mainnet already proved. It records `CANARY_AT_BATCH` / `CANARY_OLD_WRAPPER` / `CANARY_NEW_WRAPPER` in `.work/follow-run.env`.

### Acceptance criteria

- Phase A: ≥ 3 backlog bundles finalize (`rollup_status=5`, `lastFinalizedBatchIndex` advancing) with **zero local proving** — coordinator/prover pidfiles absent.
- After t1: routing assertions pass (`getVerifier(10, N-1)` → old wrapper, `getVerifier(10, N)` → new wrapper).
- Parallel period: bundles with `end_batch < N` finalize via the old wrapper from imported proofs while bundles with `end_batch >= N` finalize via the new wrapper from local proofs, both reaching `rollup_status=5` in the same window. The first ≥ N finalize is t2, the de-facto cutover.
- Rollback drill: after `canary-rollback`, `getVerifier(10, N)` routes to the old wrapper again and ≥ N bundles finalize from quarantined proofs; a subsequent `canary-upgrade` re-establishes the parallel state.
- `finalized_lag` (make follow-status) returns to ≤ 1 and stays there.

### Notes & traps

- **The imported proofs must match the forked production wrapper's digests.** They do by construction (same production system) — unless the fork point is so old that the wrapper/proofs straddle a production upgrade. Verify the production guest version first (see "Determining the Production zk Stack" above and `../docs/bundle-digest-encoding.md`).
- **Proof import never clobbers local state**: the apply rule only touches `proof IS NULL AND rollup_status <> 5` rows, and after t1 the boundary partitions the work — the new coordinator only sees unproven ≥ N bundles, the importer only applies < N.
- The quarantine table `remote_bundle_proof` is shadow-private (created by the sync script, not goose) and survives `--reset` truncations; `make follow-stop` does not remove it either. Drop it manually if you switch the shadow DB to a different purpose.
- New trap entry: TROUBLESHOOTING.md Trap 39 (proof-import mode pitfalls).
- **Docker provers**: `04-prover-up.sh --docker` (driven via `30-canary-upgrade.sh --docker-provers` / `make canary-upgrade DOCKER_PROVERS=1`) runs each prover as a `shadow-prover-<gpu>` container of `PROVER_IMAGE` with same-path work-dir mounts, host networking, `--user $(id -u)` and the SRS mounted at `$HOME/.openvm/params` inside the container (HOME is overridden so the path matches). The container's host PID is written to the usual pidfile, so `11-follow-stop.sh` / `31-canary-rollback.sh` work unchanged (plus a `docker rm -f shadow-prover-*` sweep as backstop). Prover logs: `docker logs shadow-prover-<gpu>` (the per-GPU `prover.log` file stays empty in this mode). Build the image from the checkout under test with `build/dockerfiles/prover.Dockerfile` — the same "production-style packaging" the devops `prover-image-build.yml` workflow produces in CI.

## Run Completion & Acceptance Criteria

A follow run ends after `FOLLOW_RUN_HOURS` (default 48) or on failure. The run **passes** when:

- **Lag ≤ 1** bundle sustained over the run window (monitor `finalized_lag`).
- **Zero manual intervention** over the run window (no manual SQL fixes, no daemon restarts).
- The report (`make follow-report`) answers: **kept pace?** (bundles created vs finalized), avg proof times per level (chunk/batch/bundle), and identifies bottlenecks (if any).

