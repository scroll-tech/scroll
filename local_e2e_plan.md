# Local E2E Test Plan: Prover + Coordinator

Working directory for all steps: `/home/combray/Code/scroll/scroll/tests/prover-e2e`

---

## Overview

This plan runs a full local end-to-end test of the coordinator and zkvm-prover pipeline.
It spins up a local PostgreSQL instance, injects pre-recorded block data as proving tasks,
builds and launches the coordinator API, then runs the zkvm-prover against those tasks.

The current `conf` symlink points to `sepolia-galileoV2` (confirmed by `readlink conf`).
All steps assume this staff set unless noted.

---

## Step 1: Check Prerequisites

Verify the following tools are installed and accessible:

- `docker` and `docker compose`
- `go` (for building `e2e_tool`)
- `goose` (DB migration tool — `GOOSE_CMD` defaults to `goose`)
- `nc` (netcat, used by the Makefile health check)
- `jq` (used by `coordinator/build/setup_releases.sh`)
- `wget` (used by `coordinator/build/setup_releases.sh`)
- `cargo` / Rust toolchain (for building and running the zkvm-prover)

Also verify:
- The `conf` symlink exists under `tests/prover-e2e` and points to a valid staff set directory.
  Currently: `conf -> sepolia-galileoV2`
- The staff set `.make.env` provides `BEGIN_BLOCK`, `END_BLOCK`, `SCROLL_FORK_NAME`, and `SCROLL_ZKVM_VERSION`.
  Currently (sepolia-galileoV2): `BEGIN_BLOCK=17086000`, `END_BLOCK=17086005`, `SCROLL_FORK_NAME=galileoV2`, `SCROLL_ZKVM_VERSION=v0.7.3-candidate-6`

---

## Step 2: Generate `conf/config.json` (IMPORTANT: always regenerate from template)

NEVER reuse an existing `conf/config.json` — it may be stale from a previous test run with a different setup.

Copy `conf/config.template.json` to `conf/config.json`:

```
cp conf/config.template.json conf/config.json
```

Edit `conf/config.json` as needed:
- The `l2.l2geth.endpoint` field must point to a reachable L2 RPC endpoint.
  For `sepolia-galileoV2` the default is `http://l2geth-rpc-0.sepolia.scroll.tech:8545/` (already set in template).
- The `l2.validium_mode` field: if `true`, ask the user for the `sequencer.decryption_key` (hex string WITHOUT `0x` prefix) and fill it in.
  For `sepolia-galileoV2` the default is `false`, so this is not required.

---

## Step 3: Run `make all` — Start DB and Import Block Data

From `tests/prover-e2e`, run:

```
make all
```

This target performs three sub-steps in sequence:
1. `setup_db`: Tears down any running containers (`docker compose down`), starts a fresh PostgreSQL container (`docker compose up --detach`), waits for it to accept connections on port 5432, then runs `goose up` to apply DB migrations.
   - DB URL: `postgresql://dev:dev@localhost:5432/scroll`
2. `test_tool`: Builds the `e2e_tool` binary from `../../rollup/tests/integration_tool` into `build/bin/e2e_tool`.
3. `import_data`: Runs `build/bin/e2e_tool --config conf/config.json <BEGIN_BLOCK> <END_BLOCK>` to import block data and inject chunk/batch/bundle proving tasks into the DB.
   - For `sepolia-galileoV2`: blocks 17086000–17086005.
   - The SQL pre-migration file `conf/00100_import_blocks.sql` is applied before import if present.

After this step the local DB contains:
- 4 chunks, 2 batches, 1 bundle awaiting proof (as declared in `testset.json`).

---

## Step 4: Run `make coordinator_setup` — Build and Configure Coordinator

From `tests/prover-e2e`, run:

```
make coordinator_setup
```

This runs `SCROLL_ZKVM_VERSION=... SCROLL_FORK_NAME=... make -C ../../coordinator localsetup` which:
1. Builds the `coordinator_api` binary into `coordinator/build/bin/`.
2. Copies `conf/config.template.json` to `coordinator/build/bin/conf/config.template.json`.
3. Runs `coordinator/build/setup_releases.sh` which downloads verifier assets (`verifier.bin`, `root_verifier_vk`, `openVmVk.json`) from S3 into `coordinator/build/bin/assets/` for the matching fork.

Then copies `conf/genesis.json` into `coordinator/build/bin/conf/`.

After this step, `coordinator/build/bin/conf/config.template.json` is in place.

---

## Step 5: Configure Coordinator `config.json`

Change directory to `coordinator/build/bin/`. Perform the following — NEVER reuse any existing `config.json`, always generate from template:

```
cd ../../coordinator/build/bin
cp conf/config.template.json conf/config.json
```

Edit `conf/config.json`:
- If `l2.validium_mode` is `true`: MUST ask the user for the `sequencer.decryption_key` (hex string WITHOUT `0x` prefix) and set it.
  For `sepolia-galileoV2` it is `false`, so `decryption_key` stays as `"not need"`.
- All other fields should be correct from the template for `sepolia-galileoV2`.

Return to `tests/prover-e2e` after this step.

---

## Step 6: Launch Coordinator API

From `coordinator/build/bin/`, start the coordinator API:

```
./coordinator_api
```

The coordinator API listens on port 8390 by default. Keep it running in the background for the next step. Wait until it logs that it is ready to accept connections.

Return to `tests/prover-e2e` after starting coordinator.

---

## Step 7: Configure zkvm-prover `config.json`

Change directory to `zkvm-prover/`. NEVER reuse any existing `config.json`:

```
cd ../../zkvm-prover
cp config.json.template config.json
```

Edit `config.json`:
- Set `sdk_config.coordinator.base_url` to `http://localhost:8390` (the locally running coordinator API).
- All other fields (circuit workspace paths, download base URLs) are already set in the template.

Return to `tests/prover-e2e` after this step.

---

## Step 8: Run `make test_e2e_run` (Background)

From `zkvm-prover/`, run the e2e test in the background:

```
make test_e2e_run
```

This runs:
```
cargo run --release -p prover -- --config ./config.json handle ../tests/prover-e2e/testset.json
```

The prover reads `testset.json` (which lists the chunk/batch/bundle hashes to prove), connects to the coordinator API at `http://localhost:8390`, fetches tasks, generates proofs, and submits them back. The coordinator then verifies each proof.

Because this step is long-running it MUST be run in the background.

---

## Notes and Caveats

- The `conf` symlink must exist before `make all`; if it is missing, create it: `ln -s <staff-set-dir> conf`.
- Available staff sets: `sepolia-galileoV2` (default, non-validium), `cloak-galileoV2` (validium, requires decryption key), `sepolia-galileo`, `mainnet-galileo`.
- The DB connection string is hardcoded to `postgres://dev:dev@localhost:5432/scroll` — ensure port 5432 is free.
- The `SCROLL_ZKVM_VERSION` from `.make.env` must match what was used to build verifier assets; mismatches will cause verification failures.
- If `validium_mode` is `true` (e.g. `cloak-galileoV2`), the decryption key must be obtained from the user before Step 5.
