# Build scroll in a stock Go builder container
FROM scrolltech/full-builder:go-1.17-rust-nightly-2022-08-23 as chef

FROM chef as planner
RUN --mount=target=. \
    cargo chef prepare --recipe-path /recipe.json

FROM chef as zkp-builder
COPY --from=planner /recipe.json common/libzkp/impl/recipe.json
COPY . .

RUN cd common/libzkp/impl && \
    cargo chef cook --release --recipe-path recipe.json

RUN cd common/libzkp/impl &&  \
    cargo build --release &&  \
    cp ./target/release/libzkp.a ../interface/

# Download Go dependencies
FROM scrolltech/full-builder:go-1.18-rust-nightly-2022-08-23 as base

WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
RUN go mod download -x


# Build coordinator
FROM base as builder

COPY --from=zkp-builder common/libzkp/interface /src/coordinator/verifier/lib

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /coordinator && go build -v -p 4 -o /bin/coordinator ./cmd

# Pull coordinator into a second stage deploy alpine container
FROM ubuntu:20.04

COPY --from=builder /bin/coordinator /bin/

ENTRYPOINT ["/bin/coordinator"]
