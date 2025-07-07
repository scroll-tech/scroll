FROM ubuntu:24.04 AS builder

RUN apt-get update -y && apt-get upgrade -y

# Install basic packages
RUN apt-get install build-essential curl wget git pkg-config -y
# Install dev-packages
RUN apt-get install libclang-dev libssl-dev llvm -y

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"
ENV CARGO_HOME=/root/.cargo

COPY . /src

RUN cd /src/zkvm-prover && make prover

FROM ubuntu:24.04 AS runtime

COPY --from=builder /src/target/release/prover /usr/local/bin/

ENTRYPOINT ["prover"]