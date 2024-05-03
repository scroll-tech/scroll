#!/bin/bash

# Save the root directory of the project
ROOT_DIR=$(pwd)

# Set the environment variable
export LD_LIBRARY_PATH=$ROOT_DIR/libzstd:$LD_LIBRARY_PATH

# Run module tests
cd $ROOT_DIR
go test -v -race -gcflags="-l" -ldflags="-s=false" -coverprofile=coverage.txt -covermode=atomic ./...
