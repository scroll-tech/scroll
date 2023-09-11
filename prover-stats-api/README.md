# prover-stats-api

This directory contains the `prover-stats-api` service that provides REST APIs to query the status of proving tasks assigned to the prover.

## Instructions

1. Build and start the `prover-stats-api` service.

    ```
    cd ./prover-stats-api
    make build
    ./build/bin/prover-stats --config=./conf/config.json
    ```

2. Open this URL in your browser to view the API documents.
    ```
    http://localhost:8990/swagger/index.html
    ```

## How to update the prover-stats-api docs

```
cd ./prover-stats-api
make swag
```
