# Bridge

[![Actions Status](https://scroll-tech/bridge/workflows/Continuous%20Integration/badge.svg)](https://scroll-tech/bridge/actions)
[![codecov](https://codecov.io/gh/scroll-tech/bridge/branch/master/graph/badge.svg)](https://codecov.io/gh/scroll-tech/bridge)

This repo contains the Scroll bridge.

In addition, launching the bridge will launch a separate instance of l2geth, and sets up a communication channel
between the two, over JSON-RPC sockets.

## Dependency

+ install `abigen`

``` bash
go get -u github.com/ethereum/go-ethereum
cd $GOPATH/src/github.com/ethereum/go-ethereum/
make
make devtools
```

## Build

```bash
make clean
make bridge
```

## db operation

* init, show version, rollback, check status db

```bash
# DB_DSN: db data source name
export DB_DSN="postgres://admin:123456@localhost/test_db?sslmode=disable"
# DB_DRIVER: db driver name
export DB_DRIVER="postgres"

# TEST_DB_DRIVER, TEST_DB_DSN: It is required when executing db test cases
export TEST_DB_DRIVER="postgres"
export TEST_DB_DSN="postgres://admin:123456@localhost/test_db?sslmode=disable" 

# init db
./build/bin/bridge reset [--config ./config.json]

# show db version
./build/bin/bridge version [--config ./config.json]

# rollback db
/build/bin/bridge rollback [--version version] [--config ./config.json]

# show db status
./build/bin/bridge status [--config ./config.json]

# migrate db
./build/bin/bridge migrate [--config ./config.json]
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
