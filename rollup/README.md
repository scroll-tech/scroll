# Rollup

This repo contains the Scroll rollup.

## Dependency

+ install `abigen`

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
