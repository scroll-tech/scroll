# Tests

## bridge & coordinator

### Prerequisite

+ rust
+ go1.18
+ docker

To run tests for bridge & coordinator, you would need to build some required docker images first.

```bash
make dev_docker # under repo root directory
```


### Run the tests

```bash
go test -v -race -covermode=atomic scroll-tech/bridge/...
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
```

You can also run some related tests (they are dependent to bridge & coordiantor) using
```bash
go test -v -race -covermode=atomic scroll-tech/database/...
go test -v -race -covermode=atomic scroll-tech/common/...
```


## Contracts

You can find the unit tests in [`<REPO_DIR>/contracts/src/test/`](../contracts/src/test/), and integration tests in [`<REPO_DIR>/contracts/integration-test/`](../contracts/integration-test/).
