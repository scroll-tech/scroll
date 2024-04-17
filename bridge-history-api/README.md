# bridge-history-api

This directory contains the `bridge-history-api` service that provides REST APIs to query txs interact with Scroll official bridge contracts

## Instructions
The bridge-history-api contains three distinct components

### bridgehistoryapi-db-cli

Provide init, show version, rollback, and check status services of DB
```
    cd ./bridge-history-api
    make bridgehistoryapi-db-cli
    ./build/bin/bridgehistoryapi-db-cli [command]
```

### bridgehistoryapi-fetcher

Fetch the transactions from both L1 and L2
```
    cd ./bridge-history-api
    make bridgehistoryapi-fetcher
    ./build/bin/bridgehistoryapi-fetcher
```

### bridgehistoryapi-api

provides REST APIs. Please refer to the API details below.
```
    cd ./bridge-history-api
    make bridgehistoryapi-api
    ./build/bin/bridgehistoryapi-api
```

## APIs provided by bridgehistoryapi-api

1. `/api/txs`
```
// @Summary    	 get all txs under the given address
// @Accept       plain
// @Produce      plain
// @Param        address query string true "wallet address"
// @Param        page_size query int true "page size"
// @Param        page query int true "page"
// @Success      200
// @Router       /api/txs [get]
```

2. `/api/l2/withdrawals`
```
// @Summary    	 get all L2 withdrawals under given address
// @Accept       plain
// @Produce      plain
// @Param        address query string true "wallet address"
// @Param        page_size query int true "page size"
// @Param        page query int true "page"
// @Success      200
// @Router       /api/l2/withdrawals [get]
```

3. `/api/l2/unclaimed/withdrawals`
```
// @Summary    	 get all L2 unclaimed withdrawals under the given address
// @Accept       plain
// @Produce      plain
// @Param        address query string true "wallet address"
// @Param        page_size query int true "page size"
// @Param        page query int true "page"
// @Success      200
// @Router       /api/l2/unclaimed/withdrawals [get]
```

4. `/api/txsbyhashes`
```
// @Summary    	 get txs by given tx hashes
// @Accept       plain
// @Produce      plain
// @Param        hashes query string array true "array of hashes"
// @Success      200
// @Router       /api/txsbyhashes [post]
```

## Running bridge-history-api locally

1. Pull the latest Redis image:
   ```
   docker pull redis:latest
   ```

2. Run the Redis container:
   ```
   docker run --name bridgehistoryapi-redis -d -p 6379:6379 redis:latest
   ```

3. Pull the latest PostgreSQL image:
   ```
   docker pull postgres:latest
   ```

4. Run the PostgreSQL container:
   ```
   docker run --name bridgehistoryapi-history-db -p 5444:5432 -e POSTGRES_PASSWORD=123456 -e POSTGRES_DB=test -d postgres
   ```

5. Run database migrations to initialize the tables:
   ```
   make bridgehistoryapi-db-cli
   ./build/bin/bridgehistoryapi-db-cli migrate
   ```

6. Run bridgehistoryapi-fetcher:
   ```
   make bridgehistoryapi-fetcher
   ./build/bin/bridgehistoryapi-fetcher
   ```

7. Run bridgehistoryapi-api:
   ```
   make bridgehistoryapi-api
   ./build/bin/bridgehistoryapi-api
   ```

The endpoints provided in [./conf/config.json](./conf/config.json) are all public endpoints and have rate limits.

For production usage:

- For L1 endpoints, utilizing a service provider's free tier should suffice.
- For L2 endpoints, consider [running a Scroll L2geth node](https://www.notion.so/scrollzkp/Mainnet-Deployment-Overview-c193b12cd8894e30a4ba77f6e97109a0) and using the exposed HTTP port.
