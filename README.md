# Scroll Monorepo

[![rollup](https://github.com/scroll-tech/scroll/actions/workflows/rollup.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/rollup.yml)
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
├── <a href="./prover">prover</a>: Prover client that runs proof generation for zkEVM circuit and aggregation circuit
├── <a href="./rollup">rollup</a>: Rollup-related services
├── <a href="https://github.com/scroll-tech/scroll-contracts.git">scroll-contracts</a>: solidity code for Scroll L1 bridge and rollup contracts and L2 bridge and pre-deployed contracts.
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

## Unit Tests

Run the tests using the following commands:

```bash
go test -v -race -covermode=atomic scroll-tech/rollup/...
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
go test -v -race -covermode=atomic scroll-tech/database/...
go test -v -race -covermode=atomic scroll-tech/common/...
```

## License

Scroll Monorepo is licensed under the [MIT](./LICENSE) license.
