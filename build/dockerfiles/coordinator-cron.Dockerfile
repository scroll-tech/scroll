# Download Go dependencies
FROM scrolltech/go-alpine-builder:1.20 as base

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

# Build coordinator
FROM base as builder
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/coordinator/cmd/cron/ && go build -v -p 4 -o /bin/coordinator_cron

# Pull coordinator into a second stage deploy alpine container
FROM alpine:latest
COPY --from=builder /bin/coordinator_cron /bin/
WORKDIR /app
ENTRYPOINT ["coordinator_cron"]