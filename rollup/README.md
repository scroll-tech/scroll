# Rollup

This directory contains the three essential rollup services for the Scroll chain:
- Gas Oracle (<a href="./cmd/gas_oracle/">gas_oracle</a>): monitors the L1 and L2 gas price and sends transactions to update the gas price oracle contracts on L1 and L2.
- Rollup Relayer (<a href="./cmd/rollup_relayer/">rollup_relayer</a>): consists of three components: chunk and batch proposer and a relayer.
    - The chunk and batch proposer proposes new chunks and batches that sends Commit Transactions for data availability and Finalize Transactions for proof verification and state finalization.

## Dependency

1. `abigen`

``` bash
go install -v github.com/scroll-tech/go-ethereum/cmd/abigen
```

2. `solc`

Ensure you install the version of solc required by [MockBridge.sol](./mock_bridge/MockBridge.sol#L2) (e.g., 0.8.24). See https://docs.soliditylang.org/en/latest/installing-solidity.html

## Build

```bash
make clean
make mock_abi
make rollup_bins
```

## Start

(Note: make sure you use different private keys for different senders in config.json.)

```bash
./build/bin/gas_oracle --config ./conf/config.json
./build/bin/rollup_relayer --config ./conf/config.json
```
