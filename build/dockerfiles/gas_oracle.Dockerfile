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

# Build gas_oracle
FROM base as builder

RUN apt-get -qq update && apt-get -qq install -y wget

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/rollup/cmd/gas_oracle/ && go build -v -p 4 -o /bin/gas_oracle

# Pull gas_oracle into a second stage deploy alpine container
FROM ubuntu:20.04

RUN apt-get -qq update && apt-get -qq install -y wget

COPY --from=builder /bin/gas_oracle /bin/
WORKDIR /app
ENTRYPOINT ["gas_oracle"]