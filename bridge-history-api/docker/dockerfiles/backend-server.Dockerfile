# Download Go dependencies
FROM golang:1.20-alpine3.16 as base

WORKDIR /src
COPY go.mod* ./
COPY ./go.* ./backend/
RUN go mod download -x

# Build backend-server
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/cmd/backend_server && go build -v -p 4 -o /bin/backend-server

# Pull backend-server into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/backend-server /bin/

ENTRYPOINT ["backend-server"]