# bridge-history-api

This directory contains the `bridge-history-api` service that provides REST APIs to query txs interact with Scroll official bridge contracts

## Instructions
The bridge-history-api contains 3 different components

If want to run `bridgehistoryapi-db-cli`， which connects the DB and provide several operations(migrate, reset...)
1. Build and run the `bridgehistoryapi-db-cli`
```
    cd ./bridge-history-api
    make bridgehistoryapi-db-cli
    ./build/bin/bridgehistoryapi-db-cli [command]
```

If want to run `bridgehistoryapi-cross-msg-fetcher`, which connects the DB and fetches txs from both l1 and l2:
2. Build and start the `bridgehistoryapi-cross-msg-fetcher` service.
```
    cd ./bridge-history-api
    make bridgehistoryapi-cross-msg-fetcher
    ./build/bin/bridgehistoryapi-cross-msg-fetcher
```

If want to run `bridgehistoryapi-server`, which connected to DB and provides REST APIs:
3. Build and start the `bridgehistoryapi-server` service.
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
// @Param        address   query  string  true  "wallet address"
// @Success      200
// @Router       https://localhost:8080/api/txs?address=0x0000000000000000000000000000000000000000&page_size=10&page=1 [get]
```

2. `/txsbyhashes`
```
// @Summary    	 get txs by given tx hashes
// @Accept       plain
// @Produce      plain
// @Param        hashes  query  string array  true  "array of hashes list"
// @Success      200  
// @Router       https://localhost:8080/api/txsbyhashes [post]
// @CallBody     {"txs":["0x5536519194bab05602cf1a1c84c297a4e917b8d65d00c7766f5978c2e585d128"]}
```

3. `/claimable`
```
// @Summary    	 get all claimable txs under given address
// @Accept       plain
// @Produce      plain
// @Param        hashes  query string  true  "wallet address"
// @Success      200
// @Router       https://localhost:8080//api/claimable?address=0x0000000000000000000000000000000000000000&page_size=10&page=1 [get]
```

4. `/withdraw_root`
```
// @Summary    	 get withdraw_root of given batch index
// @Accept       plain
// @Produce      plain
// @Param        batch_index  query string  true  "batch_index"
// @Success      200
// @Router        https://localhost:8080//api/withdraw_root?batch_index=1 [get]
```