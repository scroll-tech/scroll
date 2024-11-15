# Download Go dependencies
FROM scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03 as base

WORKDIR /src
COPY go.mod* ./
COPY ./bridge-history-api/go.* ./
RUN go mod download -x

# Build bridgehistoryapi-fetcher
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge-history-api/cmd/fetcher && CGO_LDFLAGS="-Wl,--no-as-needed -ldl" go build -v -p 4 -o /bin/bridgehistoryapi-fetcher

# Pull bridgehistoryapi-fetcher into a second stage deploy ubuntu container
FROM ubuntu:20.04

ENV CGO_LDFLAGS="-Wl,--no-as-needed -ldl"
RUN apt update && apt install ca-certificates vim netcat-openbsd net-tools curl -y
RUN update-ca-certificates
COPY --from=builder /bin/bridgehistoryapi-fetcher /bin/
WORKDIR /app
ENTRYPOINT ["bridgehistoryapi-fetcher"]
