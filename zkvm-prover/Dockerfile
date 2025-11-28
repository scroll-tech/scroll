FROM rust:1 AS builder

COPY . .
RUN cd /scroll/zkvm-prover && make cpu_prover

FROM debian:trixie-slim AS runtime
WORKDIR app
COPY --from=builder /app/target/release/prover ./prover
ENTRYPOINT ["./prover"]
