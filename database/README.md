# DATABASE CLIENT

[![Actions Status](https://scroll-tech/bridge/workflows/Continuous%20Integration/badge.svg)](https://scroll-tech/database/actions)
[![codecov](https://codecov.io/gh/scroll-tech/bridge/branch/master/graph/badge.svg)](https://codecov.io/gh/scroll-tech/database)

This repo contains the Scroll database client.

Database client will provide init, show version, rollback, check status services

## Build

``` bash
make db_cli
```

## Usage
``` bash
# Migrate
db_cli migrate
# Reset
db_cli reset
# Status
db_cli status
# Version
db_cli version
# RollBack
db_cli rollback
```

## Test

```bash
make test
```

## db config

* db settings in config

```bash
# DB_DSN: db data source name
export DB_DSN="postgres://admin:123456@localhost/test_db?sslmode=disable"
# DB_DRIVER: db driver name
export DB_DRIVER="postgres"
```
