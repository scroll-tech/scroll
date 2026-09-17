# Prover: The scroll prover to generate zk proof for rollup data: chunk, batch and bundle

## Prepare 
Use `config.json.template` to generate `config.json` for configuring prover，often we can use the content
inside of the template file with following tuning:

+ Update the `sdk_config.coordinator.base_url` field to the url for coordinator. The mostly used urls are:
    * For sepolia testnet: `https://sepolia-coordinator.scroll.io`
    * For mainnet: `https://coordinator.scroll.io`
    * For local e2e test: `http://localhost:8390`

+ For a new task (not restart from any interrupted task before), create a new dir under `.work` for a clean database and
  update the `sdk_config.db_path` field

+ If only wish to test some specified type of task (chunk, batch or bundle), you can specify the corresponding number in 
`sdk_config.prover.supported_proof_types`:
    * `1` for chunk
    * `2` for batch
    * `3` for bundle

+ Aggregation tasks (batch/bundle) on OpenVM v2 require `circuits.<fork>.child_circuit_vks`: it maps the
  child proof type (`1` = chunk, `2` = batch) to the child circuit's VK, so the prover can load the child
  assets and enable deferred STARK verification. `config.json.template` carries the values for the current
  release; without it every batch/bundle task fails with "missing child circuit vk". The matching
  `asset_detours` entry per VK tells the prover where to download each circuit's assets
  (`app.vmexe`, `openvm.toml`, `agg_vk.bin`).

### Build prover using cpu or gpu
+ `make prover` build the prover binary making use of CUDA
+ `make prover_halo2gpu` build the prover binary with the `halo2-gpu` feature: the bundle SNARK
  (EVM proof) also runs on GPU (~seconds vs tens of minutes on CPU). Needs a 24 GB-class GPU.
+ `make prover_cpu` build the prover binary only use CPU for proving

## Run as service
Call the prover binary with `--confing <path of config file>`, the binary will run as proving service, keep pulling task from coordinator, proving the task and submit the result back to coordinator

## Run for specified tasks
A batches of task ids can be specified and let prover to handle them (even they have been handled before):
Prepare a json file like following:
```json
{
    "chunks": [
        <task id>,
        <task id>,
        ...
    ],
    "batches": [
        <task id>,
        <task id>,
        ...
    ],
    "bundles": [
        <task id>,
        <task id>,
        ...
    ]
}
```
Run prover with `handle` command, specify the json file (suppose it is `workset.json` under current dir):
`../target/release/prover --config config.json handle workset.json`

## Dumping task data
Prover can dump a task to working dir by the `dump` command, the task id and task type (chunk, batch or bundle) must be specified:
`../target/release/prover --config config.json dump <task type> <task id>`
