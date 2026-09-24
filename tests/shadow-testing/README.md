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
| [`follow/GUIDE.md`](follow/GUIDE.md) | Follow mode — bring-up, daemons, steady state, recovery, mid-run hard-switch upgrade, canary parallel-upgrade |
| [`snapshot/GUIDE.md`](snapshot/GUIDE.md) | Snapshot replay mode — step-by-step manual setup, verifier deployment, relayer dry-run |
| [`docs/TROUBLESHOOTING.md`](docs/TROUBLESHOOTING.md) | **Trap index** — all numbered traps → per-file locations |
| [`docs/COMMON-TROUBLESHOOTING.md`](docs/COMMON-TROUBLESHOOTING.md) | Mode-independent traps, pre-flight ritual, checklists, symptom table |
| [`docs/CURRENT-STACK.md`](docs/CURRENT-STACK.md) | Dated version-sensitive facts (production stack, S3 prefixes, digest encoding, codec thresholds) |
| [`docs/rds-query-rules.md`](docs/rds-query-rules.md) | Production RDS query discipline (partial indexes, `count(*)` bans) |
| [`docs/bundle-digest-encoding.md`](docs/bundle-digest-encoding.md) | Bundle verifier digest encodings per release (Montgomery vs canonical) |
| [`follow/TROUBLESHOOTING.md`](follow/TROUBLESHOOTING.md) | Follow-mode traps (poll sync, starvation, live finalization, canary) |
| [`snapshot/TROUBLESHOOTING.md`](snapshot/TROUBLESHOOTING.md) | Snapshot-replay traps (historical fork, fixed bundle range, Sepolia) |
| [`docs/contract-addresses.md`](docs/contract-addresses.md) | L1 contract addresses per network |
| `../../docs/testing_reports/` | Dated test reports (one file per run; indexed from root `AGENTS.md`) |

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

## Contributing & Documentation Conventions

When you discover a new trap or workaround, **write it back** — this toolkit's value is its accumulated failure knowledge. Follow the layering below so the corpus stays navigable as it grows.

### Knowledge layering — where new knowledge goes

| Layer | File(s) | What belongs there | What does NOT |
|---|---|---|---|
| **Runbook** | `follow/GUIDE.md`, `snapshot/GUIDE.md` | How to run a mode: procedures, commands, acceptance criteria, recovery paths | One-off incidents, version-baked addresses/digests |
| **Trap** | `docs/COMMON-TROUBLESHOOTING.md` (mode-independent), `follow/TROUBLESHOOTING.md`, `snapshot/TROUBLESHOOTING.md` (active); `docs/TRAP-ARCHIVE.md` (retired) | A failure mode with symptom → cause → fix, learned from real debugging | Procedures that never failed, plain explanations |
| **Report** | `../../docs/testing_reports/<mode>-<variant>-YYYY-MM-DD.md` | What happened in one test run: timeline, findings, acceptance table, evidence (tx hashes, digests) | Durable procedure (promote that part to runbook/trap instead) |
| **Facts** | `docs/CURRENT-STACK.md` | Version-sensitive values: production stack identity, S3 prefixes, digest encodings, codec block thresholds, sizing | Anything timeless |
| **Index** | `docs/TROUBLESHOOTING.md` | The trap registry table only | Trap content |

Rules of thumb:

- **Link, don't duplicate.** If the same fact is needed in two places, one becomes canonical and the other links to it. Duplicated prose drifts (this has already happened once — a whole Sepolia table and verifier section lived in root `AGENTS.md` beside the canonical copies).
- **Reports are immutable.** Post-run, extract anything durable into traps/runbooks and leave the report as a dated record. Reference reports from traps ("observed 2026-09-11, report …"), not the other way around.
- **Facts age; procedures don't.** If you catch yourself writing an address, digest, S3 path, or block threshold into a runbook or trap, put the value in `CURRENT-STACK.md` with a `last verified` date and link to it.

### Trap conventions

1. **Numbering is globally unique across all trap files, including the archive.** The next free number = `max(existing) + 1` (see the "next free number" line in the registry; `make doc-check` detects collisions). Numbers are never reused — a retired number stays retired.
2. **Lifecycle — every trap has a status** tracked in the registry: **Active** (body in its home file) / **Fixed** (patched upstream or in the harness) / **Checklist** (one-time setup issue absorbed into a COMMON checklist). When a fix lands: update the registry row, move the body to `docs/TRAP-ARCHIVE.md` with a `> **Status: …**` line naming the fix, and leave the number retired. References to retired traps keep resolving (doc-check includes the archive).
3. **New-trap bar**: a trap must be a **recurring failure mode or a non-obvious diagnosis**. One-time-per-machine environment issues go into the Phase 0 checklist instead; pure explanations belong in a GUIDE. When in doubt, ask whether someone will grep for this symptom again — if not, it is not a trap.
4. **Trap→hardening review**: after each significant test run (or at least quarterly), walk the Active list and ask of each trap: *"can the harness prevent this automatically?"* If yes, patch the script (a watchdog, a sanity check, a canonicalized path), verify, then retire the trap per rule 2. The Active count should trend flat or down — a forever-growing Active set means we are accumulating "traps a human must remember" instead of "failures the system prevents".
5. **Style**: `### Trap N: Short Title [both | follow mode | snapshot mode | build | tooling]` followed by **Symptom** / **Cause** / **Fix** / **Rule** (or **Diagnosis** / **Prevention** where they fit). Keep the symptom grep-able — exact error strings and selectors (`0x439cc0cd`) beat paraphrases, and the registry's symptom column is the primary search surface.
6. **Status markers**: `> **Fixed upstream / harness**: …` at the top when the root cause is patched but the trap is still live in some setups.
7. **Cross-references**: `Trap N` (unique, so no file qualifier needed), or `follow/GUIDE.md "Section Name"`. Every reference is validated by `make doc-check`.
8. **Renumbering is a last resort** and requires fixing every reference plus a note on the trap (`> Renumbered YYYY-MM-DD from "Trap N", which collided with …`).

### Report conventions

- One file per run: `<mode>-<variant>-YYYY-MM-DD.md` (e.g. `canary-parallel-upgrade-2026-09-11.md`).
- Include: mode + host + checkout, stacks-under-test table (old vs new with digests/wrappers), UTC timeline, findings (each with the fix that was applied), acceptance table with evidence (tx hashes, trace excerpts, digests).
- Update the reports index in root `AGENTS.md` when adding one.

### Hygiene check

Run `make doc-check` from `tests/shadow-testing/` before committing documentation changes — it fails on duplicate trap numbers and on `Trap N` references that resolve to zero or multiple definitions. Keep the registry's status column and totals accurate by hand (doc-check covers numbers and references, not statuses).
