# Coordinator

This repo contains the Scroll coordinator.

## Build

```bash
make clean
make coordinator
```
The built coordinator binary is in the build/bin directory.

## Test

When developing coordinator, use the following command to mock verifier results and run coordinator tests only:

```bash
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
```
Instead of using verifier/verifier.go, it will use verifier/mock.go to always return true.

Lint the files before test and commit to remote:

```bash
make lint
```

## Config

The coordinator behavior can be configured using config.json. Check the code comments under `RollerManagerConfig` in config/config.go for more detail.

## Start

* use default ports and config.json

```bash
./build/bin/coordinator --http
```

* use specified ports and config.json

```bash
./build/bin/coordinator --config ./config.json --http --http.addr localhost --http.port 8390
```

* For other usable flags, refer to `./cmd/app/flags.go`.

## Codeflow

![圖片](https://user-images.githubusercontent.com/5474709/234186392-452c638e-aada-4431-8d33-d7bbefa6e7d3.png)

### ./cmd/app/app.go

This file defines the main entry point for the coordinator application, setting up the necessary modules, and handling graceful shutdowns. Upon loading config.json file, the coordinator (`./cmd/app/app.go`) sets up and starts the HTTP and WebSocket servers using the configured ports and addresses. flags.go is used to parse the flags.
Then, it creates a new RollerManager (`./manager.go`) and starts listening.

### ./manager.go
`manager.go` calls `rollers.go` for roller management functions. In the process, `rollers.go` calls `client.go`, initialize a roller client.  For communications between roller clients and coordinator(manager), `api.go` is used.

`manager.go` uses either `verifier.go` or `mock.go`(for test purposes) to verify the proof submitted by rollers. After verification, `manager.go` will call `roller.go` to update the state of the roller, then return the result (whether the proof is successful) to the roller.

### ./api.go

This file contains the implementation of the RPC API for the coordinator(manager). The API allows roller clients to interact with the coordinator(manager) through functions such as `requestToken`, `register`, and `submitProof`.

### ./rollers.go

This file contains the logic for handling roller-specific tasks, such as assigning tasks to rollers, handling completed tasks, and managing roller metrics.

### ./client/client.go

This file contains the Client struct, which is callable on the roller side, responsible for communicating with the coordinator through RPC calls. `RequestToken`, `RegisterAndSubscribe`, and `SubmitProof` are used by rollers.go.
