#!/bin/bash

# Download .so files
sudo mkdir -p /scroll/lib/
sudo wget -O /scroll/lib/libzktrie.so https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libzktrie.so
sudo wget -O /scroll/lib/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libscroll_zstd.so

# Set the environment variable
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/scroll/lib/
export CGO_LDFLAGS="-L/scroll/lib/ -Wl,-rpath,/scroll/lib/"

# Run module tests
go test -v -race -gcflags="-l" -ldflags="-s=false" -coverprofile=coverage.txt -covermode=atomic ./...
