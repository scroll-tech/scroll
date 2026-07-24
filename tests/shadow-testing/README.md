# Shadow Testing Toolkit

Toolkit for running Scroll shadow fork tests against Anvil: a local coordinator + local prover fed by production task data, without interfering with the live system. There are two test modes, each in its own directory with its own Makefile, scripts, configs and guide.

## Choose a Mode

| Mode | What It Does | When To Use |
|------|--------------|-------------|
| **[Follow mode](follow/)** (primary) | Fork the **current** ETH mainnet state, baseline-sync the shadow DB, then follow mainnet indefinitely — poll-syncing new chunk/batch/bundle rows, proving them locally, finalizing on the fork — until a preset duration (default 48h) or a failure | **Default acceptance test** for prover/guest upgrades |
| **[Snapshot replay mode](snapshot/)** | Fork a historical block, import a fixed bundle range, prove & finalize ~N bundles | Reproducing a specific incident, debugging one bundle, Sepolia testing, targeted codec-migration checks |

## Quick Start

```bash
# Follow mode (primary acceptance test)
cd tests/shadow-testing/follow
make follow         # one-shot bring-up of the full follow-mode stack
make follow-status  # latest monitor snapshot + lag summary
make follow-report  # final report, windowed to this run (SHADOW_REPORT_START)
make follow-stop    # tear down (script supports --keep-anvil)

# Mid-run upgrade test (continuous finalization across a zk-stack upgrade):
# Phase 1 on the production release, then cut over to the new release on the
# SAME fork/DB. See follow/GUIDE.md "Mid-Run Upgrade Test".
make follow-old      # Phase 1: old stack = production release + production verifier
make follow-upgrade  # Phase 2: deploy/register new verifier, swap coordinator
                     # assets, restart provers — finalization never stops

# Snapshot replay mode (fixed bundle range)
cd tests/shadow-testing/snapshot
make docker-all CONFIG=mainnet BUNDLE_RANGE=17302:17305   # Docker (recommended)
make all CONFIG=mainnet BUNDLE_RANGE=17297:17301          # bare-metal
```

The root `Makefile` is a thin dispatcher (`make follow-up`, `make snapshot-all ...`, `make help`). `lib/` holds scripts shared by both modes (`01-setup-anvil.sh`, `03-deploy-verifier.sh`, `04-prover-up.sh`, `06-run-relayer.sh`, `anvil-utils.sh`, `sync-queue-hashes.py`, `configs/relayer.json.template`). Runtime state (`.work/` — logs, pidfiles, Anvil state, `follow-run.env`) is shared by both modes at `tests/shadow-testing/.work`.

## Documentation

| Document | What It Covers |
|----------|----------------|
| [`follow/GUIDE.md`](follow/GUIDE.md) | Follow mode — bring-up, daemons, steady state, recovery, acceptance criteria |
| [`snapshot/GUIDE.md`](snapshot/GUIDE.md) | Snapshot replay mode — step-by-step manual setup, architecture, configuration, verifier deployment, relayer dry-run |
| [`docs/COMMON-TROUBLESHOOTING.md`](docs/COMMON-TROUBLESHOOTING.md) | Mode-independent pitfalls, traps, and agent checklists |
| [`docs/bundle-digest-encoding.md`](docs/bundle-digest-encoding.md) | Bundle verifier digests: S3 files are Montgomery for v0.8.0, canonical for v0.9.0+ — wrong encoding = `VerificationFailed` |
| [`follow/TROUBLESHOOTING.md`](follow/TROUBLESHOOTING.md) | Follow-mode-specific traps (poll sync, starvation, live finalization) |
| [`snapshot/TROUBLESHOOTING.md`](snapshot/TROUBLESHOOTING.md) | Snapshot-replay-mode-specific traps (historical fork, fixed bundle range, Sepolia) |
| [`docs/contract-addresses.md`](docs/contract-addresses.md) | L1 contract addresses per network |

## Prerequisites

- [Foundry](https://book.getfoundry.sh/) (`cast`, `forge`)
- `jq`
- `docker compose` (for Docker mode)
- PostgreSQL client (`psql`)
- Access to production RDS (via IDC port-forward)

## Configurations

Each mode has its own `configs/`; copy templates and fill in secrets:

```bash
cp follow/configs/mainnet.json.template follow/configs/mainnet.json
cp follow/configs/coordinator.json.template follow/configs/coordinator.json
# Only for the mid-run upgrade test: the NEW release's config
cp follow/configs/mainnet-next.json.template follow/configs/mainnet-next.json
cp snapshot/configs/sepolia.json.template snapshot/configs/sepolia.json
# Edit the files and replace placeholders:
#   - YOUR_ALCHEMY_API_KEY   (in mainnet.json / sepolia.json fork.url)
#   - YOUR_SHADOW_DB_PASSWORD (in mainnet.json / sepolia.json db.dsn)
#   - prover.s3_base_url + prover.circuit_version + assets.assets_v2
#     (in mainnet-next.json — the NEW release under test)
```

## Contributing

When you discover a new trap or workaround, add it to `docs/COMMON-TROUBLESHOOTING.md` (mode-independent) or the per-mode `follow/TROUBLESHOOTING.md` / `snapshot/TROUBLESHOOTING.md` (structured).
