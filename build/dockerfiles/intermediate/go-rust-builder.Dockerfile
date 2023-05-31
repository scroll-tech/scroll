FROM ubuntu:20.04

RUN apt-get update && ln -fs /usr/share/zoneinfo/America/New_York /etc/localtime

# Install basic packages
RUN apt-get install build-essential curl wget git pkg-config -y

# Install dev-packages
RUN apt-get install libclang-dev libssl-dev llvm -y

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"
ENV CARGO_HOME=/root/.cargo

# Add Toolchain
RUN rustup toolchain install nightly-2022-12-10

# TODO: make this ARG
ENV CARGO_CHEF_TAG=0.1.41

RUN cargo install cargo-chef --locked --version ${CARGO_CHEF_TAG} \
    && rm -rf $CARGO_HOME/registry/

# Install Go
RUN rm -rf /usr/local/go
# for 1.17
# RUN wget https://go.dev/dl/go1.17.13.linux-amd64.tar.gz
# RUN tar -C /usr/local -xzf go1.17.13.linux-amd64.tar.gz
# for 1.19
RUN wget https://go.dev/dl/go1.19.9.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.19.9.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
