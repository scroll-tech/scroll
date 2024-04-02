# Scroll Monorepo

[![rollup](https://github.com/scroll-tech/scroll/actions/workflows/rollup.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/rollup.yml)
[![contracts](https://github.com/scroll-tech/scroll/actions/workflows/contracts.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/contracts.yml)
[![bridge-history](https://github.com/scroll-tech/scroll/actions/workflows/bridge_history_api.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/bridge_history_api.yml)
[![coordinator](https://github.com/scroll-tech/scroll/actions/workflows/coordinator.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/coordinator.yml)
[![prover](https://github.com/scroll-tech/scroll/actions/workflows/prover.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/prover.yml)
[![integration](https://github.com/scroll-tech/scroll/actions/workflows/integration.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/integration.yml)
[![codecov](https://codecov.io/gh/scroll-tech/scroll/branch/develop/graph/badge.svg?token=VJVHNQWGGW)](https://codecov.io/gh/scroll-tech/scroll)

<a href="https://scroll.io">Scroll</a> is a zkRollup Layer 2 dedicated to enhance Ethereum scalability through a bytecode-equivalent [zkEVM](https://github.com/scroll-tech/zkevm-circuits) circuit. This monorepo encompasses essential infrastructure components of the Scroll protocol. It contains the L1 and L2 contracts, the rollup node, the prover client, and the prover coordinator.

## Directory Structure

<pre>
├── <a href="./bridge-history-api/">bridge-history-api</a>: Bridge history service that collects deposit and withdraw events from both L1 and L2 chains and generates withdrawal proofs
├── <a href="./common/">common</a>: Common libraries and types
├── <a href="./coordinator/">coordinator</a>: Prover coordinator service that dispatches proving tasks to provers
├── <a href="./database">database</a>: Database client and schema definition
├── <a href="./src">l2geth</a>: Scroll execution node
├── <a href="./prover">prover</a>: Prover client that runs proof generation for zkEVM circuit and aggregation circuit
├── <a href="./rollup">rollup</a>: Rollup-related services
├── <a href="./rpc-gateway">rpc-gateway</a>: RPC gateway external repo
└── <a href="./tests">tests</a>: Integration tests
</pre>

## Contributing

We welcome community contributions to this repository. Before you submit any issues or PRs, please read the [Code of Conduct](CODE_OF_CONDUCT.md) and the [Contribution Guideline](CONTRIBUTING.md).

## Prerequisites
+ Go 1.21
+ Rust (for version, see [rust-toolchain](./common/libzkp/impl/rust-toolchain))
+ Hardhat / Foundry
+ Docker

To run the tests, it is essential to first pull or build the required Docker images. Execute the following commands in the root directory of the repository to do this:

```bash
docker pull postgres
make dev_docker
```

## Testing Rollup & Coordinator

### For Non-Apple Silicon (M1/M2) Macs

Run the tests using the following commands:

```bash
go test -v -race -covermode=atomic scroll-tech/rollup/...
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
go test -v -race -covermode=atomic scroll-tech/rollup/...
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
go test -v -race -covermode=atomic scroll-tech/database/...
go test -v -race -covermode=atomic scroll-tech/common/...
```

## Testing Contracts

You can find the unit tests in [`contracts/src/test/`](/contracts/src/test/), and integration tests in [`contracts/integration-test/`](/contracts/integration-test/).

See [`contracts`](/contracts) for more details on the contracts.

## License

Scroll Monorepo is licensed under the [MIT](./LICENSE) license.

## Unit Test Instructions
1. `git checkout 4844-contracts-unit-test-branch` off of [https://github.com/scroll-tech/scroll].
2. FYI, The unit test file is located in `./contracts`. (`contracts/ScrollChainWithBlobTest.ts`)
3. Use nodejs version 18.0.0.
4. Follow the steps in the `Dependencies` & `Build` sections in here [https://github.com/scroll-tech/scroll/tree/develop/contracts].
5. Boot up testnet sepolia separately.
6. `npx hardhat --network local run contracts/ScrollChainWithBlobTest.ts`.