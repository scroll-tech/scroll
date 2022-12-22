# Download Go dependencies
FROM scrolltech/go-alpine-builder:1.18 as base

WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
RUN go work edit -dropuse=./tests/integration-test && \
    go mod download -x

# Build db_cli
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/database/cmd && go build -v -p 4 -o /bin/db_cli

# Pull db_cli into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/db_cli /bin/

ENTRYPOINT ["db_cli"]
