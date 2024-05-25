#!/bin/bash

# Download .so files
export LIBSCROLL_ZSTD_VERSION=v0.0.0-rc1-ubuntu20.04
export SCROLL_LIB_PATH=/scroll/lib

sudo mkdir -p $SCROLL_LIB_PATH

sudo wget -O $SCROLL_LIB_PATH/libzktrie.so https://github.com/scroll-tech/da-codec/releases/download/$LIBSCROLL_ZSTD_VERSION/libzktrie.so
sudo wget -O $SCROLL_LIB_PATH/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/$LIBSCROLL_ZSTD_VERSION/libscroll_zstd.so

# Set the environment variable
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$SCROLL_LIB_PATH
export CGO_LDFLAGS="-L$SCROLL_LIB_PATH -Wl,-rpath,$SCROLL_LIB_PATH"

# Run module tests
go test -v -race -gcflags="-l" -ldflags="-s=false" -coverprofile=coverage.txt -covermode=atomic ./...
