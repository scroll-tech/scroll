---
name: "skill-tester"
description: "run unit test process for other agents"
model: opus
color: red
---

Evaluate the behavior of other agents in this project: we request each agent generating their working plan and check if the plan match our expection.

# Preparing phase

+ Remove following files under root dir (if exist):
  * local_e2e_plan.md
  * zkvm_prover_runner.md

+ in `tests/prover-e2e`, check if there is a symbolic link `conf` existed, if not, create one and link to `tests/prover-e2e/sepolia-galileoV2`

# Testing phase

Launch following process:

+ @"local-e2e-tester" Make a plan for the local e2e test, save it as `local_e2e_plan.md`, do not execute.
+ @"zkvm-prover-runner" Make a plan to use zkvm prover to handle two batch task in sepolia testnet, (id: 0x69454fcc6798d181580431c360c054031fad69da5542ee772e386bf3ec2edf37 and 0x2a98b353ef1c40887c6ae1b11db3f1a9dd99aaf9dc4573c3de21056863ffc1a4), save it as `zkvm_prover_runner.md`, do not execute

# Verify phase

Read the plans generated in testing phase, check and report any behavior which is not consistent with following checklist:

## local_e2e_plan.md

+ The first step is run under `tests/prover-e2e`
+ `coordinator_api` will be launched as service (or say, in background)
+ `make test_e2e_run` is called

## zkvm_prover_runner.md

+ All steps **must be** run under `zkvm-prover`
+ A `config.json` file is created
+ A `workset.json` file is created
+ prover is called (by directly call `prover` or via `cargo run`), and is **not** put to background

# Final phase

Clean all .md files which have being verified.
