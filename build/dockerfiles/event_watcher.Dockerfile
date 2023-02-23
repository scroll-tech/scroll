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

# Build batch_proposer
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge/cmd/event_watcher/cmd && go build -v -p 4 -o /bin/event_watcher

# Pull batch_proposer into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/event_watcher /bin/

ENTRYPOINT ["event_watcher"]
