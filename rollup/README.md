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

## Proposer Tool

The Proposer Tool replays historical blocks with custom configurations (e.g., future hardfork configs, custom chunk/batch/bundle proposer configs) to generate chunks/batches/bundles, helping test parameter changes before protocol upgrade.

You can:

1. Enable different hardforks in the genesis configuration.
2. Set custom chunk-proposer, batch-proposer, and bundle-proposer parameters.
3. Analyze resulting metrics (blob size, block count, transaction count, gas usage).

## How to run the proposer tool?

### Set the configs

1. Set genesis config to enable desired hardforks in [`proposer-tool-genesis.json`](./proposer-tool-genesis.json).
2. Set proposer config in [`proposer-tool-config.json`](./proposer-tool-config.json) for data analysis.
3. Set `start-l2-block` in the launch command of proposer-tool in [`docker-compose-proposer-tool.yml`](./docker-compose-proposer-tool.yml) to the block number you want to start from. The default is `0`, which means starting from the genesis block.

### Start the proposer tool using docker-compose

Prerequisite: an RPC URL to an archive L2 node. The default url in [`proposer-tool-config.json`](./proposer-tool-config.json) is `https://rpc.scroll.io`.

```
cd rollup
DOCKER_BUILDKIT=1 docker-compose -f docker-compose-proposer-tool.yml up -d
```

> Note: The port 5432 of database is mapped to the host machine. You can use `psql` or any db clients to connect to the database.

> The DSN for the database is `postgres://postgres:postgres@db:5432/scroll?sslmode=disable`.


### Reset env
```
docker-compose -f docker-compose-proposer-tool.yml down -v
```

If you need to rebuild the images, removing the old images is necessary. You can do this by running the following command:
```
docker images | grep rollup | awk '{print $3}' | xargs docker rmi -f
```
