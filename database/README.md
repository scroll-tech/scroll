# DATABASE CLIENT

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
