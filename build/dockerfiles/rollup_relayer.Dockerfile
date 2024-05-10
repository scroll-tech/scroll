# Download Go dependencies
FROM scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03 as base

WORKDIR /src
COPY go.work* ./
COPY ./rollup/go.* ./rollup/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./prover/go.* ./prover/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
RUN go mod download -x

# Build rollup_relayer
FROM base as builder

RUN mkdir /opt/lib
RUN apt-get -qq update && apt-get -qq install -y wget
RUN wget -O /opt/lib/libzktrie.so https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libzktrie.so
RUN wget -O /opt/lib/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libscroll_zstd.so
ENV LD_LIBRARY_PATH=/opt/lib
ENV CGO_LDFLAGS="-L/opt/lib -Wl,-rpath=/opt/lib"

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/rollup/cmd/rollup_relayer/ && go build -v -p 4 -o /bin/rollup_relayer

# Pull rollup_relayer into a second stage deploy alpine container
FROM ubuntu:20.04

RUN mkdir /opt/lib
RUN apt-get -qq update && apt-get -qq install -y wget
RUN wget -O /opt/lib/libzktrie.so https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libzktrie.so
RUN wget -O /opt/lib/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libscroll_zstd.so
ENV LD_LIBRARY_PATH=/opt/lib
ENV CGO_LDFLAGS="-L/opt/lib -Wl,-rpath=/opt/lib"

COPY --from=builder /bin/rollup_relayer /bin/
WORKDIR /app
ENTRYPOINT ["rollup_relayer"]