# Prover

This directory contains the Scroll Prover module.


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

When you need to mock prover results and run other prover tests (using [`core/mock.go`](core/mock.go) instead of [`core/prover.go`](core/prover.go)), run:

```bash
go test -tags="mock_prover" -v -race -covermode=atomic scroll-tech/prover/...
```


## Configure

The prover behavior can be configured using [`config.json`](config.json). Check the code comments of `Config` and `ProverCoreConfig` in [`config/config.go`](config/config.go) for more details.


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

