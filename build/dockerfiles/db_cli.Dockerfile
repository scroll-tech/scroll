# Download Go dependencies
FROM scrolltech/go-builder:1.18 as base

COPY database/go.* /database/
RUN cd /database && go mod download -x

# Build db_cli
FROM base as builder

COPY ./ /
RUN cd /database/cmd && go build -v -p 4 -o db_cli

# Pull db_cli into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /database/cmd/db_cli /bin/

ENTRYPOINT ["db_cli"]
