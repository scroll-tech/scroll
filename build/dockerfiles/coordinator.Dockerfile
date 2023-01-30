# Build libzkp dependency
FROM scrolltech/go-rust-builder:go-1.18-rust-nightly-2022-08-23 as chef
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
FROM scrolltech/go-rust-builder:go-1.18-rust-nightly-2022-08-23 as base
WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
COPY ./tests/integration-test/go.* ./tests/integration-test/
RUN go mod download -x


# Build coordinator
FROM base as builder
COPY . .
RUN cp -r ./common/libzkp/interface ./coordinator/verifier/lib
COPY --from=zkp-builder /app/target/release/libzkp.so ./coordinator/verifier/lib/
# RUN cd ./coordinator && go build -v -p 4 -o /bin/coordinator ./cmd
RUN cd ./coordinator && go test -c verifier/verifier_test.go && mv verifier.test /bin/ && mv verifier/lib /bin/

# Pull coordinator into a second stage deploy alpine container
FROM ubuntu:20.04

COPY ./coordinator/assets /bin/
RUN mkdir -p /bin/lib
COPY --from=builder /bin/verifier.test /bin/
COPY --from=builder /bin/lib /bin/lib


ENTRYPOINT ["/bin/verifier.test"]
