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

# Build db_cli
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/database/cmd && go build -v -p 4 -o /bin/db_cli

# Pull db_cli into a second stage deploy container
FROM ubuntu:20.04

COPY --from=builder /bin/db_cli /bin/

ENTRYPOINT ["db_cli"]
