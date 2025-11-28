FROM scrolltech/cuda-go-rust-builder:cuda-12.9.1-go-1.22.12-rust-nightly-2025-08-18 AS builder
WORKDIR /app
COPY . .
RUN cd /app/zkvm-prover && make cpu_prover

FROM debian:trixie-slim AS runtime
WORKDIR app
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && update-ca-certificates \
 && rm -rf /var/lib/apt/lists/*

ENV RUST_LOG='off,scroll_proving_sdk=info,prover=info'

COPY --from=builder /app/target/release/prover ./prover
ENTRYPOINT ["./prover"]
