# Coordinator

[![Actions Status](https://scroll-tech/coordinator/workflows/Continuous%20Integration/badge.svg)](https://scroll-tech/coordinator/actions)
[![codecov](https://codecov.io/gh/scroll-tech/coordinator/branch/master/graph/badge.svg)](https://codecov.io/gh/scroll-tech/coordinator)

This repo contains the Scroll coordinator.

## Build

```bash
make clean
make coordinator
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
./build/bin/coordinator reset [--config ./config.json]

# show db version
./build/bin/coordinator version [--config ./config.json]

# rollback db
/build/bin/coordinator rollback [--version version] [--config ./config.json]

# show db status
./build/bin/coordinator status [--config ./config.json]

# migrate db
./build/bin/coordinator migrate [--config ./config.json]
```

## Start

* use default ports and config.json

```bash
./build/bin/coordinator --http
```

* use specified ports and config.json

```bash
./build/bin/coordinator --config ./config.json --http --http.addr localhost --http.port 8290
```
