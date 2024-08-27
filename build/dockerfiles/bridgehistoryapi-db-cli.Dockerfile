# Download Go dependencies
FROM golang:1.21-alpine3.19 as base

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
RUN apk update && apk add ca-certificates
RUN update-ca-certificates
WORKDIR /app
ENTRYPOINT ["db_cli"]