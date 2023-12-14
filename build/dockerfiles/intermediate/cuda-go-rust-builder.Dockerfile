ARG CUDA_VERSION=11.7.1
ARG GO_VERSION=1.20
ARG RUST_VERSION=nightly-2022-12-10
ARG CARGO_CHEF_TAG=0.1.41

FROM nvidia/cuda:${CUDA_VERSION}-devel-ubuntu22.04
RUN apt-get update
# Install basic packages
RUN apt-get install build-essential curl wget git pkg-config --no-install-recommends -y
# Install dev-packages
RUN apt-get install libclang-dev libssl-dev cmake llvm --no-install-recommends -y
# Install related libs
RUN apt install libprocps-dev libboost-all-dev libmpfr-dev libgmp-dev --no-install-recommends -y
# Clean installed cache
RUN rm -rf /var/lib/apt/lists/*

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"
ENV CARGO_HOME=/root/.cargo

# Add Toolchain
ARG RUST_VERSION
RUN rustup toolchain install ${RUST_VERSION}
ARG CARGO_CHEF_TAG
RUN cargo install cargo-chef --locked --version ${CARGO_CHEF_TAG} \
    && rm -rf $CARGO_HOME/registry/

# Install Go
ARG GO_VERSION
RUN rm -rf /usr/local/go
RUN wget https://go.dev/dl/go${GO_VERSION}.1.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go${GO_VERSION}.1.linux-amd64.tar.gz
RUN rm go${GO_VERSION}.1.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
