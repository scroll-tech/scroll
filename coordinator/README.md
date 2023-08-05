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

The coordinator behavior can be configured using [`config.json`](config.json). Check the code comments under `ProverManager` in [`config/config.go`](config/config.go) for more details.


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

