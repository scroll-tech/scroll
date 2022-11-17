# Download Go dependencies
FROM scrolltech/go-builder:1.18 as base

COPY bridge/go.* /bridge/
RUN cd /bridge && go mod download -x

# Build bridge
FROM base as builder

COPY ./ /
RUN cd /bridge/cmd && go build -v -p 4 -o bridge

# Pull bridge into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bridge/cmd/bridge /bin/

ENTRYPOINT ["bridge"]
