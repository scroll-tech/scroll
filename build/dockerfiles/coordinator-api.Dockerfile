# Build libzkp dependency
FROM scrolltech/cuda-go-rust-builder:cuda-11.7.1-go-1.21-rust-nightly-2023-12-03 as chef
WORKDIR app

FROM chef as planner
COPY ./crates ./
COPY ./Cargo.* ./
COPY ./rust-toolchain ./
RUN cargo chef prepare --recipe-path recipe.json

FROM chef as zkp-builder
COPY ./rust-toolchain ./
COPY --from=planner /app/recipe.json recipe.json
# run scripts to get openvm-gpu
COPY ./build/dockerfiles/coordinator-api/plonky3-gpu /plonky3-gpu
COPY ./build/dockerfiles/coordinator-api/openvm-stark-gpu /openvm-stark-gpu
COPY ./build/dockerfiles/coordinator-api/openvm-gpu /openvm-gpu
COPY ./build/dockerfiles/coordinator-api/gitconfig /root/.gitconfig
COPY ./build/dockerfiles/coordinator-api/config.toml /root/.cargo/config.toml
RUN cargo chef cook --release --recipe-path recipe.json

COPY ./crates ./
COPY ./Cargo.* ./
RUN cargo build --release -p libzkp-c


# Download Go dependencies
FROM scrolltech/cuda-go-rust-builder:cuda-11.7.1-go-1.21-rust-nightly-2023-12-03 as base
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
FROM nvidia/cuda:11.7.1-runtime-ubuntu22.04
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
