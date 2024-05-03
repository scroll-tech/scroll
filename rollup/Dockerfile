ARG GO_VERSION=1.21
ARG RUST_VERSION=nightly-2023-12-03
ARG CARGO_CHEF_TAG=0.1.41

FROM ubuntu:20.04

RUN apt-get update && ln -fs /usr/share/zoneinfo/America/New_York /etc/localtime

# Install basic packages
RUN apt-get install -y build-essential
RUN apt-get install -y curl
RUN apt-get install -y wget
RUN apt-get install -y git
RUN apt-get install -y pkg-config

# Install dev-packages
RUN apt-get install -y libclang-dev
RUN apt-get install -y libssl-dev
RUN apt-get install -y llvm

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

RUN apt-get update
RUN apt-get install -y docker.io
RUN apt-get install -y docker-compose
