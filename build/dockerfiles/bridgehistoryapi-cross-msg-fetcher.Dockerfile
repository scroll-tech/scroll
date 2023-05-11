# Download Go dependencies
FROM golang:1.20-alpine3.16 as base

WORKDIR /src
COPY go.mod* ./
COPY ./bridge-history-api/go.* ./
RUN go mod download -x

# Build bridgehistoryapi-cross-msg-fetcher
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge-history-api/cmd/cross_msg_fetcher && go build -v -p 4 -o /bin/bridgehistoryapi-cross-msg-fetcher

# Pull bridgehistoryapi-cross-msg-fetcher into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/bridgehistoryapi-cross-msg-fetcher /bin/

ENTRYPOINT ["bridgehistoryapi-cross-msg-fetcher"]