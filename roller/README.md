# Roller

This repo contains the Scroll roller.

## Build
```bash
make clean
make roller
```
The built roller binary is in the build/bin directory.

## Test

TBD.

For current unit tests, run:
```bash
make roller
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./prover/lib
export CHAIN_ID=534353
go test -v ./...
```

When you need to mock prover results and run other roller tests:
```bash
go test -tags="mock_prover" -v -race -covermode=atomic scroll-tech/roller/...
```
It will use [`prover/mock.go`](prover/mock.go) instead of [`prover/prover.go`](prover/prover.go).

Lint the files before testing or committing:
```bash
make lint
```

## Configure

The roller behavior can be configured using [`config.json`](config.json). Check the code comments of `Config` and `ProverConfig` in [`config/config.go`](config/config.go), and `NewRoller` in [`roller.go`](roller.go) for more details.

## Start
* Set environment variables
```bash
export CHAIN_ID=534353 # change to correct chain_id
export RUST_MIN_STACK=100000000 
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./prover/lib:/usr/local/cuda/   # cuda only for GPU machine
```

* Using default settings in config.json  
```bash
./build/bin/roller
```

## Codeflow

### cmd/app/app.go

This file defines the main entry point for the roller application, initializes roller instances via roller.go, and handling graceful shutdowns. The roller (`cmd/app/app.go`) calls `NewRoller` with config.json parsed and cfg passed to (`roller.go`). It then starts creating new instance of the Roller via `r.Start`, and starting the main processing loop for generating proofs dispatched from the coordinator.

Multiple Rollers can be started separately, and registered to the coordinator via coordinator's API.

### cmd/app/mock_app.go

This file wrapped the mock app functions, and is used in the integration-test. (See [integration-test](../tests/integration-test/)

### roller.go

roller.go contains the core logic of the Roller, including starting the Roller, registering with the Coordinator, handling tasks from the Coordinator, and proving loop. The Roller interacts with `prover.go` and `stack.go` to perform its functions.

`NewRoller`: A constructor function for creating a new Roller instance. It initializes the Roller with the provided configuration, loads or creates a private key, initializes the Stack and Prover instances, and sets up a client connection to the Coordinator.

`Start`: Starts the Roller by registering with the Coordinator. Contains `Register`, `HandleCoordinator`, and `ProveLoop`.

* `Register` constructs, signs an `AuthMsg` object, and send it to the coordinator. A token is then returned from the coordinator, which is used as challenge-response in `AuthMsg`, to authenticate the Roller in subsequent communications with the Coordinator. The AuthMsg object contains the roller's identity information, such as its name, public key, timestamp, and an one-time token. The last request from the Roller is `RegisterAndSubscribe`, to register and subscribe to the coordinator for receiving tasks. Related functions like `RequestToken` and `RegisterAndSubscribe` are defined in [`../coordinator/client/client.go`](../coordinator/client/client.go).

* `HandleCoordinator` and `ProveLoop` are then started listening in separate goroutines, ready to handle incoming tasks from the Coordinator.

* `HandleCoordinator` handles incoming tasks from the Coordinator by pushing them onto the Stack using the `Push` method of the `store.Stack` instance. When the subscription returns an error, the Roller will attempt to re-register and re-subscribe to the Coordinator in `mustRetryCoordinator`.

* `ProveLoop` pops tasks from the Stack and sends them to the Prover for processing.
Calling relationship:
`ProveLoop()`
    ->`prove()`
        ->`stack.Peek()`
        ->`stack.UpdateTimes()`
        ->`prover.Prove()`
        ->`stack.Delete()`
        ->`signAndSubmitProof()`

Refer to relative functions in stack, prover, or client module for more detail.

### prover/prover.go

prover.go is a part of the roller package, and it focuses on the go implementation of the Prover struct, which is responsible for proving and generating proofs using the provided tasks from the coordinator. It handles interactions with the rust-prover library via FFI. Refer to `create_agg_proof_multi` in [`../common/libzkp/impl/src/prove.rs`](../common/libzkp/impl/src/prove.rs) for more detail.

### store/stack.go

stack.go is a part of the roller package, and it's responsible for managing the task storage and retrieval for the Roller. It uses [BBolt database](https://github.com/etcd-io/bbolt) to store the tasks and provides various functions like `Push`, `Peek`, `Delete`, and `UpdateTimes`, to interact with the stored tasks.

### roller_metrics.go

roller_metrics.go is called in [`../coordinator/manager.go`](../coordinator/manager.go), and is used to collect metrics from the Roller.