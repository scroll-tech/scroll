# ProverE2E: A new e2e test tool to setup a local environment for testing coordinator and prover.

It contains data from some blocks in a specified testnet, and helps to generate a series of chunks/batches/bundles from these blocks, filling the DB for the coordinator, so an e2e test (from chunk to bundle) can be run completely local

## Prepare
link the staff dir as "conf" from one of the dir with staff set, currently we have following staff sets:
+ sepolia: with blocks from scroll sepolia
+ cloak-xen: with blocks from xen sepolia, which is a cloak network

## Test
1. run `make all` under `tests/prover-e2e`, it would launch a postgreSql db in local docker container, which is ready to be used by coordinator (include some chunks/batches/bundles waiting to be proven)
2. setup assets by run `make coordinator_setup`
3. come into `coordinator/build/bin` for following steps:
  + rename `conf/config.template.json` as `conf/config.json`
  + if the `l2.validium_mode` is set to true in `config.json`, the `sequencer.decryption_key` must be set
  + launch `coordinator_api` service by executing the file
4. come into `zkvm-prover` for following steps:
  + copy `config.template.json` to `config.json`, 
  + set the `sdk_config.coordinator.base_url` field in `config.json`, so zkvm prover would connect with the locally launched coordinator api,
    for common case the url is `http://localhost:8390` (the default listening port of coordinator api)
  + launch `make test_e2e_run`, which would specific prover run locally, connect to the local coordinator api service according to the `config.json`, and prove all tasks being injected to db in step 1.