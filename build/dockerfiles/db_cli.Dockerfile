# Download Go dependencies
FROM scrolltech/go-alpine-builder:1.21 as base

WORKDIR /src
COPY go.work* ./
COPY ./rollup/go.* ./rollup/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
RUN go mod download -x

# Build db_cli
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/database/cmd && CGO_LDFLAGS="-ldl" go build -v -p 4 -o /bin/db_cli

# Pull db_cli into a second stage deploy ubuntu container
FROM ubuntu:20.04
ENV CGO_LDFLAGS="-ldl"
COPY --from=builder /bin/db_cli /bin/
WORKDIR /app
ENTRYPOINT ["db_cli"]
