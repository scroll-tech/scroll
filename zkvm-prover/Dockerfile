FROM rust:1.85

RUN rustup toolchain install nightly-2025-02-14-x86_64-unknown-linux-gnu
RUN rustup component add rust-src --toolchain nightly-2025-02-14-x86_64-unknown-linux-gnu

WORKDIR /app
RUN git clone --branch refactor/scroll_mono --single-branch --depth 1 https://github.com/scroll-tech/zkvm-prover.git zkvm-circuits

ADD common/types-rs /app/common/types-rs
ADD zkvm-prover /app/zkvm-prover
ADD Cargo.* /app/
ADD rust-toolchain /app/
ADD build/.cargo /app/zkvm-circuits/.cargo
WORKDIR /app/zkvm-circuits

RUN cargo fetch

ENTRYPOINT ["/app/zkvm-circuits/build-guest-actions-entrypoint.sh"]
