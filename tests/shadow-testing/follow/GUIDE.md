# Follow Mode Guide — Shadow Coordinator + Prover Testing

This guide documents the **follow mode** of the shadow coordinator + local prover environment: fork the current ETH mainnet state and follow mainnet bundle production in real time. This is the **default acceptance test for prover/guest upgrades**.

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

Follow mode forks the **current** ETH mainnet state with Anvil, baseline-syncs the shadow DB from the mainnet read replica, then keeps following mainnet indefinitely — poll-syncing new chunk/batch/bundle rows, proving them with local GPU provers, and finalizing bundles on the fork — until a preset duration (`FOLLOW_RUN_HOURS`, default 48) or a failure. It answers the question a snapshot cannot: *can this prover fleet keep pace with real mainnet bundle production?*

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
6. **Relayer** — start the rollup relayer with local proposers **disabled** (`l2_config.{chunk,batch,bundle}_proposer_config.disable = true`): the relayer then only commits/finalizes what the DB contains, instead of synthesizing bundles at an unrealistic rate.
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

## Run Completion & Acceptance Criteria

A follow run ends after `FOLLOW_RUN_HOURS` (default 48) or on failure. The run **passes** when:

- **Lag ≤ 1** bundle sustained over the run window (monitor `finalized_lag`).
- **Zero manual intervention** over the run window (no manual SQL fixes, no daemon restarts).
- The report (`make follow-report`) answers: **kept pace?** (bundles created vs finalized), avg proof times per level (chunk/batch/bundle), and identifies bottlenecks (if any).

