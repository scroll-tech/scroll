# Build

FROM scrolltech/rust-alpine-builder:nightly-2022-08-23 AS chef
WORKDIR app

FROM chef AS planner
COPY . .
RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder
COPY --from=planner /app/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json
COPY . .
RUN cargo build --release

# Release

FROM alpine:3.15

ENV OPEN_API_ADDR=$open_api_addr
ENV RUN_MODE="production"

RUN mkdir -p /root/config
COPY --from=builder /app/.env /root/
COPY --from=builder /app/config/ /root/config/
COPY --from=builder /app/target/release/rollup_explorer /bin/

WORKDIR /root

ENTRYPOINT ["rollup_explorer"]
