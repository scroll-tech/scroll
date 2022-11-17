# Download Go dependencies
FROM scrolltech/go-builder:1.18 as base

COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
RUN go mod download -x

# Build coordinator
FROM base as builder

COPY ./ /
RUN cd /coordinator/cmd && go build -v -p 4 -o coordinator

# Pull coordinator into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/cmd/coordinator /bin/

ENTRYPOINT ["coordinator"]
