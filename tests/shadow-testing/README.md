# Shadow Testing Toolkit

One-command toolkit for running Scroll shadow fork tests against Anvil.

## Quick Start

### Docker (Recommended)

```bash
cd tests/shadow-testing
make docker-all CONFIG=mainnet BUNDLE_RANGE=17302:17305
```

### Bare-Metal

```bash
cd tests/shadow-testing
make all CONFIG=mainnet BUNDLE_RANGE=17297:17301
```

See `make help` for all targets.

## Documentation

| Document | What It Covers |
|----------|----------------|
| [`docs/GUIDE.md`](docs/GUIDE.md) | Full setup guide — step-by-step manual setup, architecture, configuration |
| [`docs/TROUBLESHOOTING.md`](docs/TROUBLESHOOTING.md) | Structured pitfalls, traps, and agent checklists |
| [`docs/CONTRACTS.md`](docs/contract-addresses.md) | L1 contract addresses per network |

## Directory Structure

```
configs/          # JSON config templates (copy and edit)
scripts/          # Numbered pipeline scripts (01-setup-anvil.sh …)
docs/             # Documentation
states/           # Anvil state files (gitignored)
.work/            # Runtime logs, pid files (gitignored)
```

## Prerequisites

- [Foundry](https://book.getfoundry.sh/) (`cast`, `forge`)
- `jq`
- `docker compose` (for Docker mode)
- PostgreSQL client (`psql`)
- Access to production RDS (via IDC port-forward)

## Configurations

Copy templates and fill in secrets:

```bash
cp configs/mainnet.json.template configs/mainnet.json
cp configs/sepolia.json.template configs/sepolia.json
cp configs/coordinator.json.template configs/coordinator.json
# Edit the files and replace placeholders:
#   - YOUR_ALCHEMY_API_KEY   (in mainnet.json / sepolia.json fork.url)
#   - YOUR_SHADOW_DB_PASSWORD (in mainnet.json / sepolia.json db.dsn)
```

## How It Works

The pipeline has three phases:

1. **Environment Setup** (`make env` or `make docker-env`)  
   Start Anvil fork, import bundle data from production RDS, reset status.

2. **Proving** (`make prove` or `make docker-prove`)  
   Start coordinator + provers, wait for proofs to complete.

3. **Finalization** (`make finalize` or `make docker-finalize`)  
   Start relayer, wait for on-chain finalization.

## Contributing

When you discover a new trap or workaround, add it to `docs/TROUBLESHOOTING.md` (structured).
