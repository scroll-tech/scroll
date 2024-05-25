ARG LIBSCROLL_ZSTD_VERSION=v0.0.0-rc1-ubuntu20.04
ARG SCROLL_LIB_PATH=/scroll/lib

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

ARG LIBSCROLL_ZSTD_VERSION
ARG SCROLL_LIB_PATH

RUN mkdir -p $SCROLL_LIB_PATH

RUN apt-get -qq update && apt-get -qq install -y wget

RUN wget -O $SCROLL_LIB_PATH/libzktrie.so https://github.com/scroll-tech/da-codec/releases/download/$LIBSCROLL_ZSTD_VERSION/libzktrie.so
RUN wget -O $SCROLL_LIB_PATH/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/$LIBSCROLL_ZSTD_VERSION/libscroll_zstd.so

ENV LD_LIBRARY_PATH=$SCROLL_LIB_PATH
ENV CGO_LDFLAGS="-L$SCROLL_LIB_PATH -Wl,-rpath,$SCROLL_LIB_PATH"

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/rollup/cmd/rollup_relayer/ && go build -v -p 4 -o /bin/rollup_relayer

# Pull rollup_relayer into a second stage deploy alpine container
FROM ubuntu:20.04

ARG LIBSCROLL_ZSTD_VERSION
ARG SCROLL_LIB_PATH

RUN mkdir -p $SCROLL_LIB_PATH

RUN apt-get -qq update && apt-get -qq install -y wget

RUN wget -O $SCROLL_LIB_PATH/libzktrie.so https://github.com/scroll-tech/da-codec/releases/download/$LIBSCROLL_ZSTD_VERSION/libzktrie.so
RUN wget -O $SCROLL_LIB_PATH/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/$LIBSCROLL_ZSTD_VERSION/libscroll_zstd.so

ENV LD_LIBRARY_PATH=$SCROLL_LIB_PATH
ENV CGO_LDFLAGS="-L$SCROLL_LIB_PATH -Wl,-rpath,$SCROLL_LIB_PATH"

COPY --from=builder /bin/rollup_relayer /bin/
WORKDIR /app
ENTRYPOINT ["rollup_relayer"]