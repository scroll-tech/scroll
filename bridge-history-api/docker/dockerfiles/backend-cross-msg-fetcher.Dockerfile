# Download Go dependencies
FROM golang:1.20-alpine3.16 as base

WORKDIR /src
COPY go.mod* ./
COPY ./go.* ./backend/
RUN go mod download -x

# Build backend-cross-msg-fetcher
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/cmd/cross_msg_fetcher && go build -v -p 4 -o /bin/backend-cross-msg-fetcher

# Pull backend-cross-msg-fetcher into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/backend-cross-msg-fetcher /bin/

ENTRYPOINT ["backend-cross-msg-fetcher"]