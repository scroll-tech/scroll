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

### bridgehistoryapi-cross-msg-fetcher

Fetch the transactions from both l1 and l2
```
    cd ./bridge-history-api
    make bridgehistoryapi-cross-msg-fetcher
    ./build/bin/bridgehistoryapi-cross-msg-fetcher
```

### bridgehistoryapi-server

provides REST APIs. Please refer to the API details below.
```
    cd ./bridge-history-api
    make bridgehistoryapi-server
    ./build/bin/bridgehistoryapi-server
```

## APIs provided by bridgehistoryapi-server

assume `bridgehistoryapi-server` listening on `https://localhost:8080`
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

2. `/txsbyhashes`
```
// @Summary    	 get txs by given tx hashes
// @Accept       plain
// @Produce      plain
// @Param        hashes  query  string array  true  "array of hashes list"
// @Success      200  
// @Router       /api/txsbyhashes [post]
```

3. `/claimable`
```
// @Summary    	 get all claimable txs under given address
// @Accept       plain
// @Produce      plain
// @Param        address query string true "wallet address"
// @Param        page_size query int true "page size"
// @Param        page query int true "page"
// @Success      200
// @Router       /api/claimable [get]
```

4. `/withdraw_root`
```
// @Summary    	 get withdraw_root of given batch index
// @Accept       plain
// @Produce      plain
// @Param        batch_index  query string  true  "batch_index"
// @Success      200
// @Router       /api/withdraw_root [get]
```