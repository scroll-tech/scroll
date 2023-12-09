# bridge-history-api

This directory contains the `bridge-history-api` service that provides REST APIs to query txs interact with Scroll official bridge contracts

## Instructions
The bridge-history-api contains three distinct components

### bridgehistoryapi-db-cli

Provide init, show version, rollback, check status services of DB
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

assume `bridgehistoryapi-api` listening on `https://localhost:8080`
can change this port thru modify `config.json`

1. `/txs`
```
// @Summary    	 get all txs under given address
// @Accept       plain
// @Produce      plain
// @Param        address query string true "wallet address"
// @Param        page_size query int true "page size"
// @Param        page query int true "page"
// @Success      200
// @Router       /api/txs [get]
```

2. `/withdrawals`
```
// @Summary    	 get all L2 withdrawals under given address
// @Accept       plain
// @Produce      plain
// @Param        address query string true "wallet address"
// @Param        page_size query int true "page size"
// @Param        page query int true "page"
// @Success      200
// @Router       /api/withdrawals [get]
```

3. `/claimablewithdrawals`
```
// @Summary    	 get all L2 claimable withdrawals under given address
// @Accept       plain
// @Produce      plain
// @Param        address query string true "wallet address"
// @Param        page_size query int true "page size"
// @Param        page query int true "page"
// @Success      200
// @Router       /api/claimablewithdrawals [get]
```

4. `/txsbyhashes`
```
// @Summary    	 get txs by given tx hashes
// @Accept       plain
// @Produce      plain
// @Param        hashes query string array true "array of hashes"
// @Success      200
// @Router       /api/txsbyhashes [post]
```
