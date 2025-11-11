# Download Go dependencies
FROM scrolltech/go-rust-builder:go-1.22.12-rust-nightly-2025-02-14 as base
WORKDIR /src
COPY go.work* ./
COPY ./rollup/go.* ./rollup/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
RUN go mod download -x


# Build coordinator proxy
FROM base as builder
COPY . .
RUN cd ./coordinator && CGO_LDFLAGS="-Wl,--no-as-needed -ldl" make coordinator_proxy && mv ./build/bin/coordinator_proxy /bin/coordinator_proxy

# Pull coordinator proxy into a second stage deploy ubuntu container
FROM ubuntu:20.04
ENV CGO_LDFLAGS="-Wl,--no-as-needed -ldl"
RUN apt update && apt install vim netcat-openbsd net-tools curl jq -y
COPY --from=builder /bin/coordinator_proxy /bin/
RUN /bin/coordinator_proxy --version
WORKDIR /app
ENTRYPOINT ["/bin/coordinator_proxy"]
