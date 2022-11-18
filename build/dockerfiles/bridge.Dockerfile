# Download Go dependencies
FROM scrolltech/go-builder:1.18 as base

WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
RUN go mod download -x

# Build bridge
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge/cmd && go build -v -p 4 -o /bin/bridge

# Pull bridge into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/bridge /bin/

ENTRYPOINT ["bridge"]
