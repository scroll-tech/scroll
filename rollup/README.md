# Rollup

This directory contains the three essential rollup services for the Scroll chain:
- Event Watcher (<a href="./cmd/event_watcher/">event_watcher</a>): watches the events emitted from the L1 and L2 contracts and updates the event database.
- Gas Oracle (<a href="./cmd/gas_oracle/">gas_oracle</a>): monitors the L1 and L2 gas price and sends transactions to update the gas price oracle contracts on L1 and L2.
- Rollup Relayer (<a href="./cmd/rollup_relayer/">rollup_relayer</a>): consists of three components: chunk and batch proposer and a relayer.
    - The chunk and batch proposer proposes new chunks and batches that sends Commit Transactions for data availability and Finalize Transactions for proof verification and state finalization.

## Dependency

Install `abigen`

``` bash
go install -v github.com/scroll-tech/go-ethereum/cmd/abigen
```

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
