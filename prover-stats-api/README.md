# prover-stats-api

## how to get the prover-stats-api docs

### 1. start the prover-stats-api server

```
cd ./prover-stats-api
make build
./prover-stats --config=./conf/config.json
```

you will get server run log
```
Listening and serving HTTP on :8990
```

### 2. browse the documents

open this documents in your browser
```
http://localhost:8990/swagger/index.html
```

## how to update the prover-stats-api docs

```
cd ./prover-stats-api
make swag
```
