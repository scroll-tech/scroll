# Build libzkp dependency
FROM scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03 as chef
WORKDIR app

FROM chef as planner
COPY ./common/libzkp/impl/ .
RUN cargo chef prepare --recipe-path recipe.json

FROM chef as zkp-builder
COPY ./common/libzkp/impl/rust-toolchain ./
COPY --from=planner /app/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json

COPY ./common/libzkp/impl .
RUN cargo build --release


# Download Go dependencies
FROM scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03 as base
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
RUN cp -r ./common/libzkp/interface ./coordinator/internal/logic/verifier/lib
COPY --from=zkp-builder /app/target/release/libzkp.so ./coordinator/internal/logic/verifier/lib/
RUN cd ./coordinator && CGO_LDFLAGS="-Wl,--no-as-needed -ldl" make coordinator_api_skip_libzkp && mv ./build/bin/coordinator_api /bin/coordinator_api && mv internal/logic/verifier/lib /bin/

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
