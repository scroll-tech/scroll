# Use the latest Rust image as the base
FROM --platform=linux/amd64 rust:latest AS builder
# Install dependencies for building Go and Rust projects
RUN apt-get update && apt-get install -y build-essential make git
# Install Go version 1.21 or above
RUN wget https://golang.org/dl/go1.21.10.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.21.10.linux-amd64.tar.gz && \
    rm go1.21.10.linux-amd64.tar.gz

# Set Go environment variables
ENV PATH="/usr/local/go/bin:${PATH}"
# Set the working directory
WORKDIR /snarkify
# Copy source code from the host to the container
COPY prover/ .
COPY common/version/version.go version.go
# Run make snarkify to build the project
RUN make snarkify
# Use a slim variant for the runtime stage to reduce size
FROM --platform=linux/amd64 ubuntu:22.04
# Copy the build artifact from the build stage
COPY --from=builder /snarkify/target/release/snarkify /usr/local/bin/snarkify
COPY --from=builder /snarkify/lib/libzktrie.so /usr/local/lib/libzktrie.so
# Set environment variables
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH \
    SCROLL_PROVER_ASSETS_DIR=/snarkify-data/assets,/snarkify-data/assets \
    RUST_MIN_STACK=100000000
# Set the entrypoint for the container
ENTRYPOINT ["snarkify"]