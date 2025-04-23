# Download Go dependencies
FROM scrolltech/go-rust-builder:go-1.22-rust-nightly-2023-12-03 as base

WORKDIR /src
COPY go.work* ./
COPY ./rollup/go.* ./rollup/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
RUN go mod download -x

# Build proposer_tool
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/rollup/cmd/proposer_tool/ && CGO_LDFLAGS="-ldl" go build -v -p 4 -o /bin/proposer_tool

# Pull proposer_tool into a second stage deploy ubuntu container
FROM ubuntu:20.04

RUN apt update && apt install vim netcat-openbsd net-tools curl ca-certificates -y

ENV CGO_LDFLAGS="-ldl"

COPY --from=builder /bin/proposer_tool /bin/
WORKDIR /app
ENTRYPOINT ["proposer_tool"]
