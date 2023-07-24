# Roller

This directory contains the Scroll prover (aka "roller") module.


## Build
```bash
make clean
make prover
```
The built prover binary is in the build/bin directory.


## Test

Make sure to lint before testing (or committing):

```bash
make lint
```

For current unit tests, run:

```bash
make prover
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./prover/lib
export CHAIN_ID=534353 # for Scroll Alpha
go test -v ./...
```

When you need to mock prover results and run other prover tests (using [`prover/mock.go`](prover/mock.go) instead of [`prover/prover.go`](prover/prover.go)), run:

```bash
go test -tags="mock_prover" -v -race -covermode=atomic scroll-tech/prover/...
```


## Configure

The prover behavior can be configured using [`config.json`](config.json). Check the code comments of `Config` and `ProverConfig` in [`config/config.go`](config/config.go), and `NewRoller` in [`roller.go`](roller.go) for more details.


## Start

1. Set environment variables:

```bash
export CHAIN_ID=534353 # change to correct chain ID
export RUST_MIN_STACK=100000000
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./prover/lib
```

2. Start the module using settings from config.json:

```bash
./build/bin/prover
```


# Flow

## `cmd/app/app.go`

This file defines the main entry point for the prover application, initializes prover instances via roller.go, and handles graceful shutdowns. The prover (`cmd/app/app.go`) calls `NewRoller` with config.json parsed and cfg passed to `roller.go`. It then starts creating new instances of provers via `r.Start` and starts the main processing loop for generating proofs dispatched from the coordinator.

Multiple prover can be started separately and registered with the coordinator via its API.


## `cmd/app/mock_app.go`

This file wrapps mock app functions and is used in the [integration test](../tests/integration-test/).


## `roller.go`

This file contains the logic of the `roller`, including starting it, registering with the coordinator, handling tasks from the coordinator, and running the proving loop. The `roller` interacts with `prover` and `stack` to perform its functions.

`NewRoller`: A constructor function for creating a new `Roller` instance. It initializes it with the provided configuration, loads or creates a private key, initializes the `Stack` and `Prover` instances, and sets up a client connection to the coordinator.

`Start`: Starts the prover and registers it with the coordinator. It contains `Register`, `HandleCoordinator` and `ProveLoop`:

* `Register` constructs and signs an `AuthMsg` object, and sends it to the coordinator. A token is then returned from the coordinator that is used as challenge-response in `AuthMsg` in order to authenticate the prover in subsequent communication with the coordinator. The `AuthMsg` object contains prover identity information, such as its name, public key, timestamp, and a one-time token. The last request from the prover is `RegisterAndSubscribe`, to register and subscribe to the coordinator for receiving tasks. Related functions like `RequestToken` and `RegisterAndSubscribe` are defined in [`../coordinator/client/client.go`](../coordinator/client/client.go). `HandleCoordinator` and `ProveLoop` are then started and listen in separate goroutines, ready to handle incoming tasks from the coordinator.

* `HandleCoordinator` handles incoming tasks from the coordinator by pushing them onto the stack using the `Push` method of the `store.Stack` instance. If the subscription returns an error, the prover will attempt to re-register and re-subscribe to the coordinator in `mustRetryCoordinator`.

* `ProveLoop` pops tasks from the stack and sends them to the prover for processing. Call order is the following: `ProveLoop()` -> `prove()` -> `stack.Peek()` -> `stack.UpdateTimes()` -> `prover.Prove()` -> `stack.Delete()` -> `signAndSubmitProof()`.

Refer to the functions in `stack`, `prover` and `client` modules for more detail.


## `prover/prover.go`

This file focuses on the go implementation of the `Prover` struct which is responsible for generating proofs from tasks provided by the coordinator. It handles interactions with the rust-prover library via FFI. Refer to `create_agg_proof_multi` in [`../common/libzkp/impl/src/prove.rs`](../common/libzkp/impl/src/prove.rs) for more detail.


## `store/stack.go`
This file is responsible for managing task storage and retrieval for the prover. It uses a [BBolt database](https://github.com/etcd-io/bbolt) to store tasks and provides various functions like `Push`, `Peek`, `Delete` and `UpdateTimes` in order to interact with them.


## `roller_metrics.go`

This file is called from [`../coordinator/roller_metrics.go`](../coordinator/roller_metrics.go) and is used to collect metrics from the prover.
