# Follow Mode (Primary)

Follow mode forks the ETH mainnet state from **FOLLOW_FORK_HOURS_BACK hours in the past** (default 5h), baseline-syncs the shadow DB from the mainnet read replica, then first **catches up** the resulting backlog (all bundles committed-but-not-finalized at the fork block plus everything mainnet produced since) and then **follows** mainnet indefinitely — poll-syncing new chunk/batch/bundle rows, proving them with local GPU provers, and finalizing on the fork — for a preset window (default 48h). It is the **default acceptance test for prover/guest upgrades**: it answers whether the prover fleet can catch up a backlog *and* keep pace with real mainnet bundle production. `FOLLOW_FORK_HOURS_BACK=0` forks at the current tip (no backlog). The final report splits metrics into catch-up and steady-state phases.

## Quick Start

```bash
cd tests/shadow-testing/follow
make follow         # one-shot bring-up: baseline sync → Anvil fork → verifier → coordinator → provers → relayer → daemons
make follow-status  # latest monitor snapshot + finalize lag
make follow-report  # throughput report for this run (SHADOW_REPORT_START)
make re-fork        # recover from a fork desync (re-fork at latest block)
make follow-stop    # tear down (script supports --keep-anvil)
```

Full guide: [`GUIDE.md`](GUIDE.md). Pitfalls/traps: [`./TROUBLESHOOTING.md`](./TROUBLESHOOTING.md) (follow mode) and [`../docs/COMMON-TROUBLESHOOTING.md`](../docs/COMMON-TROUBLESHOOTING.md) (mode-independent). Runtime state lives in the shared `../.work/`.
