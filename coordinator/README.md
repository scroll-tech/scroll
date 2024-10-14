# Coordinator

This directory contains the Scroll Coordinator module.


## Prerequisites

See [monorepo prerequisites](../README.md#prerequisites).


## Build

```bash
make clean
make coordinator_api
make coordinator_cron
```
The built coordinator binary is in the `build/bin` directory.


## Test

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

The coordinator behavior can be configured using [`conf/config.json`](conf/config.json). Check the code comments under `ProverManager` in [`internal/config/config.go`](internal/config/config.go) for more details.


## Start

* Using default ports and config.json:
```bash
./build/bin/coordinator_api --http
./build/bin/coordinator_cron 
```

* Using manually specified ports and config.json:
```bash
./build/bin/coordinator_api --config ./config.json --http --http.addr localhost --http.port 8390
./build/bin/coordinator_cron --config ./config.json 
```

* For other flags, refer to [`cmd/api/app/flags.go`](cmd/api/app/flags.go).

