# Snapshot Replay Mode

Snapshot replay forks a **historical** ETH mainnet block, imports a fixed bundle range from production RDS, and proves + finalizes ~N bundles on the fork. Use it for incident reproduction, single-bundle debugging, Sepolia testing, and targeted codec-migration checks — anything where following live mainnet is the wrong tool.

## Quick Start

```bash
cd tests/shadow-testing/snapshot
make docker-all CONFIG=mainnet BUNDLE_RANGE=17302:17305   # Docker (recommended)
make all CONFIG=mainnet BUNDLE_RANGE=17297:17301          # bare-metal: env → prove → finalize
make sepolia-all CONFIG=sepolia BUNDLE_RANGE=<start:end>  # Sepolia pipeline
make help                                                 # all targets
```

Full guide: [`GUIDE.md`](GUIDE.md). Pitfalls/traps: [`./TROUBLESHOOTING.md`](./TROUBLESHOOTING.md) (snapshot replay mode) and [`../docs/COMMON-TROUBLESHOOTING.md`](../docs/COMMON-TROUBLESHOOTING.md) (mode-independent). Runtime state lives in the shared `../.work/`.
