# Download Go dependencies
FROM scrolltech/go-builder:1.18 as base

COPY coordinator/go.* /coordinator/
RUN cd /coordinator && go mod download -x

# Build coordinator
FROM base as builder

COPY ./ /
RUN cd /coordinator/cmd && go build -v -p 4 -o coordinator

# Pull coordinator into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/cmd/coordinator /bin/

ENTRYPOINT ["coordinator"]
