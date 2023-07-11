# Download Go dependencies
FROM golang:1.20-alpine3.16 as base

WORKDIR /src
COPY ./bridge-history-api/go.* ./
RUN go mod download -x

# Build db_cli
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge-history-api/cmd/db_cli && go build -v -p 4 -o /bin/db_cli

# Pull db_cli into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/db_cli /bin/

ENTRYPOINT ["db_cli"]