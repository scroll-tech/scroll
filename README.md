[![codecov](https://codecov.io/gh/scroll-tech/scroll/branch/develop/graph/badge.svg?token=VJVHNQWGGW)](https://codecov.io/gh/scroll-tech/scroll)

# Scroll Monorepo

## Prerequisites
+ Go 1.19
+ Rust (for version, see [rust-toolchain](./common/libzkp/impl/rust-toolchain))
+ Hardhat / Foundry
+ Docker

To run the tests, it is essential to first pull or build the required Docker images. Execute the following commands in the root directory of the repository to do this:

```bash
docker pull postgres
make dev_docker
```

## Testing Bridge & Coordinator

### For Non-Apple Silicon (M1/M2) Macs

Run the tests using the following commands:

```bash
go test -v -race -covermode=atomic scroll-tech/bridge/...
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
go test -v -race -covermode=atomic scroll-tech/database/...
go test -v -race -covermode=atomic scroll-tech/common/...
```

### For Apple Silicon (M1/M2) Macs

To run tests on Apple Silicon Macs, build and execute the Docker image as outlined below:

#### Build a Docker Image for Testing

Use the following command to build a Docker image:

```bash
make build_test_docker
```

This command builds a Docker image named `scroll_test_image` using the Dockerfile found at `./build/dockerfiles/local_test.Dockerfile`.

#### Run Docker Image

After the image is built, run a Docker container from it:

```bash
make run_test_docker
```

This command runs a Docker container named `scroll_test_container` from the `scroll_test_image` image. The container uses the host network and has access to the Docker socket and the current directory.

Once the Docker container is running, execute the tests using the following commands:

```bash
go test -v -race -covermode=atomic scroll-tech/bridge/...
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
go test -v -race -covermode=atomic scroll-tech/database/...
go test -v -race -covermode=atomic scroll-tech/common/...
```

## Testing Contracts

You can find the unit tests in [`<REPO_DIR>/contracts/src/test/`](/contracts/src/test/), and integration tests in [`<REPO_DIR>/contracts/integration-test/`](/contracts/integration-test/).

For more details on contracts, see [`/contracts`](/contracts).
