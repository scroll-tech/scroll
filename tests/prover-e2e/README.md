## A new e2e test tool to setup a local environment for testing coordinator and prover.

It contains data from some blocks in scroll sepolia, and helps to generate a series of chunks/batches/bundles from these blocks, filling the DB for the coordinator, so an e2e test (from chunk to bundle) can be run completely local

Steps:
1. run `make all` under `tests/prover-e2e`, it would launch a postgreSql db in local docker container, which is ready to be used by coordinator (include some chunks/batches/bundle waiting to be proven)
2. download circuit assets with `download-release.sh` script in `zkvm-prover`
3. generate the verifier stuff corresponding to the downloaded assets by `make gen_verifier_stuff` in `zkvm-prover`
4. setup `config.json` and `genesis.json` for coordinator, copy the generated verifier stuff in step 3 to the directory which coordinator would load them
5. build and launch `coordinator_api` service locally
6. setup the `config.json` for zkvm prover to connect with the locally launched coordinator api
7. in `zkvm-prover`, launch `make test_e2e_run`, which would specific prover run locally, connect to the local coordinator api service according to the `config.json`, and prove all tasks being injected to db in step 1.