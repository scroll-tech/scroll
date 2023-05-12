# Scroll Monorepo

[![Contracts](https://github.com/scroll-tech/scroll/actions/workflows/contracts.yaml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/contracts.yaml) [![Bridge](https://github.com/scroll-tech/scroll/actions/workflows/bridge.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/bridge.yml) [![Coordinator](https://github.com/scroll-tech/scroll/actions/workflows/coordinator.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/coordinator.yml) [![Database](https://github.com/scroll-tech/scroll/actions/workflows/database.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/database.yml) [![Common](https://github.com/scroll-tech/scroll/actions/workflows/common.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/common.yml) [![Roller](https://github.com/scroll-tech/scroll/actions/workflows/roller.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/roller.yml)

## Prerequisites
+ Go 1.18
+ Rust (for version, see [rust-toolchain](./common/libzkp/impl/rust-toolchain))
+ Hardhat / Foundry
+ Docker

To run the tests, it is essential to first pull or build the required Docker images. Execute the following commands in the root directory of the repository to do this:

```bash
docker pull postgres
make dev_docker
```

## Testing Bridge & Coordinator

### Run Tests

```bash
go test -v -race -covermode=atomic scroll-tech/bridge/...
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
```

### Testing Bridge & Coordinator on Apple Silicon (M1/M2) Macs

To conduct tests on Apple Silicon Macs, follow these steps:

Ensure Docker is installed on your system.
Open a terminal and navigate to the directory where this README.md is located.

#### Build a Docker Image for Testing

Firstly, you need to build a Docker image. You can do this by running the following command:

```bash
make build_test_docker
```

This command will build a Docker image using the Dockerfile located at `./build/dockerfiles/local_test.Dockerfile`. The image will be named `scroll_test_image`.

#### Run Docker Image

After the image has been built, you can run a Docker container from it:

```bash
make run_test_docker
```

This command will run a Docker container named `scroll_test_container` from the `scroll_test_image` image. The container will use the host network, and it will have access to the Docker socket and the current directory.

This setup provides a testing environment compatible with Apple Silicon Macs.

## Testing Database & Common

```bash
go test -v -race -covermode=atomic scroll-tech/database/...
go test -v -race -covermode=atomic scroll-tech/common/...
```

## Testing Contracts

You can find the unit tests in [`<REPO_DIR>/contracts/src/test/`](../contracts/src/test/), and integration tests in [`<REPO_DIR>/contracts/integration-test/`](../contracts/integration-test/).

For a more comprehensive doc for contracts, see [`docs/`](./docs/contracts).
