# Build libzkp dependency
FROM scrolltech/go-1.22.12-rust-nightly-2025-02-14 as chef
WORKDIR app

FROM chef as planner
COPY ./crates/ ./crates/
COPY ./Cargo.* ./
COPY ./rust-toolchain ./
RUN cargo chef prepare --recipe-path recipe.json

FROM chef as zkp-builder
COPY ./rust-toolchain ./
COPY --from=planner /app/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json
COPY ./crates/ ./crates/
COPY ./Cargo.* ./
COPY .git .git
RUN cargo build --release -p libzkp-c


# Download Go dependencies
FROM scrolltech/go-1.22.12-rust-nightly-2025-02-14 as base
WORKDIR /src
COPY go.work* ./
COPY ./rollup/go.* ./rollup/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
RUN go mod download -x


# Build coordinator
FROM base as builder
COPY . .
COPY --from=zkp-builder /app/target/release/libzkp.so ./coordinator/internal/logic/libzkp/lib/
RUN cd ./coordinator && CGO_LDFLAGS="-Wl,--no-as-needed -ldl" make coordinator_api && mv ./build/bin/coordinator_api /bin/coordinator_api
RUN mv coordinator/internal/logic/libzkp/lib /bin/

# Pull coordinator into a second stage deploy ubuntu container
FROM ubuntu:20.04
ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/src/coordinator/internal/logic/verifier/lib
ENV CGO_LDFLAGS="-Wl,--no-as-needed -ldl"
# ENV CHAIN_ID=534353
RUN apt update && apt install vim netcat-openbsd net-tools curl jq -y
RUN mkdir -p /src/coordinator/internal/logic/verifier/lib
COPY --from=builder /bin/lib /src/coordinator/internal/logic/verifier/lib
COPY --from=builder /bin/coordinator_api /bin/
RUN /bin/coordinator_api --version
WORKDIR /app
ENTRYPOINT ["/bin/coordinator_api"]
