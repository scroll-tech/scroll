#!/bin/bash

# Download .so files
wget https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libzktrie.so
wget https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libscroll_zstd.so

# Set the environment variable
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$(PWD)
export CGO_LDFLAGS="-L$(PWD) -Wl,-rpath=$(PWD)"

# Run module tests
go test -v -race -gcflags="-l" -ldflags="-s=false" -coverprofile=coverage.txt -covermode=atomic ./...
