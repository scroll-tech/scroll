# Download Go dependencies
FROM scrolltech/go-alpine-builder:1.18 as base

WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
COPY ./tests/integration-test/go.* ./tests/integration-test/
RUN go mod download -x

# Build message_relayer
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge/multibin/message_relayer/cmd && go build -v -p 4 -o /bin/message_relayer

# Pull message_relayer into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/message_relayer /bin/

ENTRYPOINT ["message_relayer"]
