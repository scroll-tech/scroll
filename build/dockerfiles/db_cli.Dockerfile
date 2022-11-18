# Download Go dependencies
FROM scrolltech/go-builder:1.18 as base

COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
COPY ./tests/integration-test/go.* ./tests/integration-test/
RUN go mod download -x

# Build db_cli
FROM base as builder

COPY ./ /
RUN cd /database/cmd && go build -v -p 4 -o db_cli

# Pull db_cli into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /database/cmd/db_cli /bin/

ENTRYPOINT ["db_cli"]
