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

# Build gas_oracle
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/rollup/cmd/gas_oracle/ && CGO_LDFLAGS="-ldl" go build -v -p 4 -o /bin/gas_oracle

# Pull gas_oracle into a second stage deploy ubuntu container
FROM ubuntu:20.04

RUN apt update && apt install ca-certificates -y

ENV CGO_LDFLAGS="-ldl"

COPY --from=builder /bin/gas_oracle /bin/
WORKDIR /app
ENTRYPOINT ["gas_oracle"]
