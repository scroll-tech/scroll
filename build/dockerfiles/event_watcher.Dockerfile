# Download Go dependencies
FROM scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03 as base

WORKDIR /src
COPY go.work* ./
COPY ./rollup/go.* ./rollup/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./prover/go.* ./prover/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
RUN go mod download -x

# Build event_watcher
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/rollup/cmd/event_watcher/ && go build -v -p 4 -o /bin/event_watcher

# Pull event_watcher into a second stage deploy alpine container
FROM ubuntu:20.04

COPY --from=builder /bin/event_watcher /bin/
WORKDIR /app
ENTRYPOINT ["event_watcher"]