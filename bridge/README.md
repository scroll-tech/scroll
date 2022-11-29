# Bridge

This repo contains the Scroll bridge.

In addition, launching the bridge will launch a separate instance of l2geth, and sets up a communication channel
between the two, over JSON-RPC sockets.

Something we should pay attention is that all private keys inside sender instance cannot be duplicated.

## Dependency

+ install `abigen`

``` bash
go install -v github.com/scroll-tech/go-ethereum/cmd/abigen
```

## Build

```bash
make clean
make bridge
```

## DB config

* db settings in config

```bash
# DB_DSN: db data source name
export DB_DSN="postgres://admin:123456@localhost/test_db?sslmode=disable"
# DB_DRIVER: db driver name
export DB_DRIVER="postgres"
```
## Start
* use default ports and config.json

```bash
./build/bin/bridge --http
```

* use specified ports and config.json

```bash
./build/bin/bridge --config ./config.json --http --http.addr localhost --http.port 8290
```
