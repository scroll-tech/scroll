# Download Go dependencies
FROM golang:1.20-alpine3.16 as base

WORKDIR /src
COPY go.mod* ./
COPY ./bridge-history-api/go.* ./
RUN go mod download -x

# Build bridgehistoryapi-server
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge-history-api/cmd/backend_server && go build -v -p 4 -o /bin/bridgehistoryapi-server

# Pull bridgehistoryapi-server into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/bridgehistoryapi-server /bin/

ENTRYPOINT ["bridgehistoryapi-server"]