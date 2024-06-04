# Download Go dependencies
FROM amd64/golang:1.21 as base

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

# Build rollup_relayer
FROM base as builder

RUN apt-get -qq update && apt-get -qq install -y wget

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/rollup/cmd/rollup_relayer/ && go build -v -p 4 -o /bin/rollup_relayer

# Pull rollup_relayer into a second stage deploy alpine container
FROM ubuntu:20.04

RUN apt-get -qq update && apt-get -qq install -y wget

COPY --from=builder /bin/rollup_relayer /bin/
WORKDIR /app
ENTRYPOINT ["rollup_relayer"]