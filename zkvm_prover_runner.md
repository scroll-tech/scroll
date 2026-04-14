# zkvm Prover: Run Two Batch Tasks on Sepolia Testnet

All steps must be run from the `zkvm-prover/` directory.

## Target Tasks

Two batch tasks to prove:

- `0x69454fcc6798d181580431c360c054031fad69da5542ee772e386bf3ec2edf37`
- `0x2a98b353ef1c40887c6ae1b11db3f1a9dd99aaf9dc4573c3de21056863ffc1a4`

---

## Step 1: Prepare the workset file

Create (or overwrite) `workset.json` in `zkvm-prover/` with the two batch IDs:

```json
{
  "chunks": [],
  "batches": [
    "0x69454fcc6798d181580431c360c054031fad69da5542ee772e386bf3ec2edf37",
    "0x2a98b353ef1c40887c6ae1b11db3f1a9dd99aaf9dc4573c3de21056863ffc1a4"
  ],
  "bundles": []
}
```

> Note: The file `workset.json` already exists in `zkvm-prover/` with these exact IDs. Overwrite it unconditionally — do NOT inspect the existing content before writing.

---

## Step 2: Prepare `config.json`

Copy `config.json.template` to `config.json` (overwrite unconditionally — do NOT inspect any pre-existing `config.json`):

```bash
cp config.json.template config.json
```

Then edit `config.json` with the following values:

| Field | Value |
|---|---|
| `sdk_config.coordinator.base_url` | `https://sepolia-coordinator.scroll.io` |
| `sdk_config.prover.supported_proof_types` | `[2]` (batch only) |
| `sdk_config.db_path` | `.work/db_batch_sepolia_<today_date>` (a fresh path, e.g. `.work/db_batch_sepolia_20260413`) |

The `circuits` section can remain as-is from the template (includes `feynman`, `galileo`, and `galileoV2`).

Example resulting `config.json`:

```json
{
  "sdk_config": {
    "prover_name_prefix": "test-prover",
    "keys_dir": ".work",
    "coordinator": {
      "base_url": "https://sepolia-coordinator.scroll.io",
      "retry_count": 10,
      "retry_wait_time_sec": 10,
      "connection_timeout_sec": 1800
    },
    "prover": {
      "supported_proof_types": [2],
      "circuit_version": "v0.13.1"
    },
    "health_listener_addr": "127.0.0.1:10080",
    "db_path": ".work/db_batch_sepolia_20260413"
  },
  "circuits": {
    "feynman": {
      "workspace_path": ".work/feynman",
      "base_url": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/feynman/"
    },
    "galileo": {
      "workspace_path": ".work/galileo",
      "base_url": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/galileo/"
    },
    "galileoV2": {
      "base_url": "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/galileov2/",
      "workspace_path": ".work/galileo"
    }
  }
}
```

---

## Step 3: Run the prover with the `handle` command

The Makefile's `test_e2e_run` target reads `E2E_HANDLE_SET` (default: `../tests/prover-e2e/testset.json`). Since we want to use our `workset.json`, override it:

```bash
make test_e2e_run E2E_HANDLE_SET=./workset.json
```

This expands to:

```
cargo run --release -p prover -- --config ./config.json handle ./workset.json
```

The prover will:
1. Connect to `https://sepolia-coordinator.scroll.io`
2. Register itself and authenticate
3. Request and prove the two batch tasks by their IDs
4. Submit the proofs back to the coordinator

**Run this step in the background** so it does not block the terminal.

---

## Notes

- No local coordinator or Docker setup is needed — the prover connects directly to the Sepolia testnet coordinator.
- The `supported_proof_types: [2]` restricts the prover to batch tasks only, which matches the two IDs.
- A fresh `db_path` is used to avoid any state conflict with previous runs.
- Circuit assets will be downloaded automatically by the prover on first run into `.work/galileo` etc.
- The `handle` command forces re-proving even if the task has been handled before, making it suitable for targeted reruns.
