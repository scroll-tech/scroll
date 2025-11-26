## A new e2e test tool to setup a local environment for testing coordinator and prover.

It contains data from some blocks in a specified testnet, and helps to generate a series of chunks/batches/bundles from these blocks, filling the DB for the coordinator, so an e2e test (from chunk to bundle) can be run completely local

Prepare:
link the staff dir as "conf" from one of the dir with staff set, currently we have following staff sets:
+ sepolia: with blocks from scroll sepolia forking
+ galileo: with blocks from scroll galileo forking
+ cloak-xen: with blocks from xen sepolia, which is a cloak network

Steps:
1. run `make all` under `tests/prover-e2e`, it would launch a postgreSql db in local docker container, which is ready to be used by coordinator (include some chunks/batches/bundles waiting to be proven)
2. setup assets by run `make coordinator_setup`, `SCROLL_ZKVM_VERSION` must be sepcified, and if we do e2e test for other forking than `Galileo`, `SCROLL_FORK_NAME` is also required, example:
~~~
   SCROLL_FORK_NAME=feynman SCROLL_ZKVM_VERSION=v0.7.0 make coordinator_setup
~~~
3. in `coordinator/build/bin/conf`, update necessary items in `config.template.json` and rename it as `config.json`
4. build and launch `coordinator_api` service locally
5. setup the `config.json` for zkvm prover to connect with the locally launched coordinator api:
  + set the `sdk_config.coordinator.base_url` field into "http://localhost:8390",
6. in `zkvm-prover`, launch `make test_e2e_run`, which would specific prover run locally, connect to the local coordinator api service according to the `config.json`, and prove all tasks being injected to db in step 1.