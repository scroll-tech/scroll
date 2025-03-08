# Download Go dependencies
FROM scrolltech/go-rust-builder:go-1.22-rust-nightly-2023-12-03 as base

WORKDIR /src
COPY ./bridge-history-api/go.* ./
RUN go mod download -x

# Build db_cli
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge-history-api/cmd/db_cli && CGO_LDFLAGS="-Wl,--no-as-needed -ldl" go build -v -p 4 -o /bin/db_cli

# Pull db_cli into a second stage deploy ubuntu container
FROM ubuntu:20.04
ENV CGO_LDFLAGS="-ldl"
COPY --from=builder /bin/db_cli /bin/
WORKDIR /app
ENTRYPOINT ["db_cli"]