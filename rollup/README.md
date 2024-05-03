# Rollup

This directory contains the three essential rollup services for the Scroll chain:
- Event Watcher (<a href="./cmd/event_watcher/">event_watcher</a>): watches the events emitted from the L1 and L2 contracts and updates the event database.
- Gas Oracle (<a href="./cmd/gas_oracle/">gas_oracle</a>): monitors the L1 and L2 gas price and sends transactions to update the gas price oracle contracts on L1 and L2.
- Rollup Relayer (<a href="./cmd/rollup_relayer/">rollup_relayer</a>): consists of three components: chunk and batch proposer and a relayer.
    - The chunk and batch proposer proposes new chunks and batches that sends Commit Transactions for data availability and Finalize Transactions for proof verification and state finalization.

## Dependency

1. `abigen`

``` bash
go install -v github.com/scroll-tech/go-ethereum/cmd/abigen
```

2. `solc`

See https://docs.soliditylang.org/en/latest/installing-solidity.html

## Build

```bash
make clean
make mock_abi
make rollup_bins
```

## Start

(Note: make sure you use different private keys for different senders in config.json.)

```bash
./build/bin/event_watcher --config ./config.json
./build/bin/gas_oracle --config ./config.json
./build/bin/rollup_relayer --config ./config.json
```

## libzstd

### Building `libscroll_zstd.so` File.

Follow these steps to build the `.so` file:

1. Build and enter the container:
    ```
    docker build -t my-dev-container --platform linux/amd64 .
    docker run -it --rm -v "$(PWD):/workspace" -w /workspace my-dev-container
    ```

2. Change directory to rs:
    ```
    cd libzstd
    ```

3. Build libzstd:
    ```
    export CARGO_NET_GIT_FETCH_WITH_CLI=true
    make libzstd
    ```

### Running unit tests

Follow these steps to run unit tests, in the repo's root dir:

1. Build and enter the container:
    ```
    docker run -it --rm --network=host -v /var/run/docker.sock:/var/run/docker.sock -v "$(PWD):/workspace" -w /workspace -e HOST_PATH=$(PWD) my-dev-container
    ```

2. Set the directory for shared libraries:
    ```
    export LD_LIBRARY_PATH=${PWD}/rollup/libzstd:$LD_LIBRARY_PATH
    ```

3. Execute the unit tests:
    ```
    cd rollup
    go test -v -race ./...
    ```
