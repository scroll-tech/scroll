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

* use default ports and config.json.

```bash
./build/bin/event_watcher --http
./build/bin/gas_oracle --http
./build/bin/rollup_relayer --http
```

* use specified ports and config.json

```bash
./build/bin/event_watcher --config ./config.json --http --http.addr localhost --http.port 8290
./build/bin/gas_oracle --config ./config.json --http --http.addr localhost --http.port 8290
./build/bin/rollup_relayer --config ./config.json --http --http.addr localhost --http.port 8290
```
