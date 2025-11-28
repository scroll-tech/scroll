FROM scrolltech/cuda-go-rust-builder:cuda-12.9.1-go-1.22.12-rust-nightly-2025-08-18 AS builder
WORKDIR /app
COPY . .
RUN cd /app/zkvm-prover && make cpu_prover

FROM debian:trixie-slim AS runtime
WORKDIR app
COPY --from=builder /app/target/release/prover ./prover
ENTRYPOINT ["./prover"]
