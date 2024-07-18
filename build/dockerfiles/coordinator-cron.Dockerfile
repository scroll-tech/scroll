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
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/coordinator/cmd/cron/ && CGO_LDFLAGS="-Wl,--no-as-needed -ldl" go build -v -p 4 -o /bin/coordinator_cron

# Pull coordinator into a second stage deploy ubuntu container
FROM ubuntu:20.04

ENV CGO_LDFLAGS="-Wl,--no-as-needed -ldl"

COPY --from=builder /bin/coordinator_cron /bin/
WORKDIR /app
ENTRYPOINT ["coordinator_cron"]
