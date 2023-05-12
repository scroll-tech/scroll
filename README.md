# Scroll Monorepo

[![Contracts](https://github.com/scroll-tech/scroll/actions/workflows/contracts.yaml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/contracts.yaml) [![Bridge](https://github.com/scroll-tech/scroll/actions/workflows/bridge.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/bridge.yml) [![Coordinator](https://github.com/scroll-tech/scroll/actions/workflows/coordinator.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/coordinator.yml) [![Database](https://github.com/scroll-tech/scroll/actions/workflows/database.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/database.yml) [![Common](https://github.com/scroll-tech/scroll/actions/workflows/common.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/common.yml) [![Roller](https://github.com/scroll-tech/scroll/actions/workflows/roller.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/roller.yml)

## Prerequisites
+ go1.18
+ rust (for version, see [rust-toolchain](./common/libzkp/impl/rust-toolchain))
+ hardhat / foundry

## Testing

### Testing on Apple Silicon (M1/M2) Macs

For testing on Apple Silicon Macs, follow these steps:

1. Ensure Docker is installed on your system.
2. Open a terminal and navigate to the directory where this README.md is located.
3. Build a Docker image for testing with:

```bash
make test_docker
```

This command will build and run a Docker container using the Dockerfile located at `./build/dockerfiles/local_test.Dockerfile`.

The container will have the name `my_scroll_test_image`, and it will provide a compatible testing environment for Apple Silicon Macs.


---

For a more comprehensive doc, see [`docs/`](./docs).
