# Download Go dependencies
FROM golang:1.21-alpine3.19 as base

WORKDIR /src
COPY ./bridge-history-api/go.* ./
RUN go mod download -x

# Build db_cli
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge-history-api/cmd/db_cli && CGO_LDFLAGS="-ldl" go build -v -p 4 -o /bin/db_cli

# Pull db_cli into a second stage deploy ubuntu container
FROM ubuntu:20.04
ENV CGO_LDFLAGS="-ldl"
COPY --from=builder /bin/db_cli /bin/
WORKDIR /app
ENTRYPOINT ["db_cli"]