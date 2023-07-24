# Download Go dependencies
FROM scrolltech/go-alpine-builder:1.18 as base

WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./prover/go.* ./prover/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
RUN go mod download -x

# Build rollup_relayer
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge/cmd/rollup_relayer/ && go build -v -p 4 -o /bin/rollup_relayer

# Pull rollup_relayer into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/rollup_relayer /bin/

ENTRYPOINT ["rollup_relayer"]