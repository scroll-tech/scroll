# Download Go dependencies
FROM scrolltech/go-alpine-builder:1.19 as base

WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./prover-stats-api/go.* ./prover-stats-api/
COPY ./prover/go.* ./prover/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
RUN go mod download -x

# Build msg_relayer
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge/cmd/msg_relayer/ && go build -v -p 4 -o /bin/msg_relayer

# Pull msg_relayer into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/msg_relayer /bin/

ENTRYPOINT ["msg_relayer"]