# Coordinator

This directory contains the Scroll Coordinator module.


## Prerequisites

See [monorepo prerequisites](../README.md#prerequisites).


## Build

```bash
make clean
make coordinator
```
The built coordinator binary is in the `build/bin` directory.


## Test

**Note:** Our test code may not directly support Apple Silicon (M1/M2) chips. However, we've provided a Docker-based solution for local testing on M1/M2 Macs. Please refer to the [Local Testing on M1/M2 Mac](../README.md#local-testing-on-m1m2-mac) section in the main README for details. After preparing the environment, you can then proceed with the testing.

When developing the coordinator, use the following command to mock verifier results and run coordinator tests:

```bash
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
```
Instead of using verifier/verifier.go, it will use verifier/mock.go to always return true.

Lint the files before testing or committing:

```bash
make lint
```


## Configure

The coordinator behavior can be configured using [`config.json`](config.json). Check the code comments under `ProverManagerConfig` in [`config/config.go`](config/config.go) for more details.


## Start

* Using default ports and config.json:
```bash
./build/bin/coordinator --http
```

* Using manually specified ports and config.json:
```bash
./build/bin/coordinator --config ./config.json --http --http.addr localhost --http.port 8390
```

* For other flags, refer to [`cmd/app/flags.go`](cmd/app/flags.go).


## Codeflow

### cmd/app/app.go

This file defines the main entry point for the coordinator application, setting up the necessary modules, and handling graceful shutdowns. Upon loading config.json file, the coordinator (`cmd/app/app.go`) sets up and starts the HTTP and WebSocket servers using the configured ports and addresses. `flags.go` is used to parse the flags. Then, it creates a new `ProverManager` (`manager.go`) and starts listening.

### manager.go

`manager.go` calls `provers.go` for prover (aka "prover") management functions. In the process, `provers.go` calls `client.go`, initializing a prover client.  For communication between prover clients and the coordinator manager, `api.go` is used.

`manager.go` uses either `verifier.go` or `mock.go` (for development/testing purposes) to verify the proofs submitted by provers. After verification, `manager.go` will call `prover.go` to update the state of the prover, and then return the result (whether the proof verification process was successful) to the prover.

### api.go

This file contains the implementation of the RPC API for the coordinator manager. The API allows prover clients to interact with the coordinator manager through functions such as `requestToken`, `register`, and `submitProof`.

### provers.go

This file contains the logic for handling prover-specific tasks, such as assigning tasks to provers, handling completed tasks, and managing prover metrics.

### client/client.go

This file contains the `Client` struct that is callable on the prover side and responsible for communicating with the coordinator through RPC. `RequestToken`, `RegisterAndSubscribe`, and `SubmitProof` are used by `provers.go`.
