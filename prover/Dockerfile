# The build step is skpped, one should build on the host first
# Use the NVIDIA CUDA runtime image for the runtime stage
FROM nvidia/cuda:12.3.0-base-ubuntu22.04

WORKDIR /snarkify

# Copy the build artifact from the build stage
COPY snarkify /usr/local/bin/snarkify
COPY libzktrie.so /usr/local/lib/libzktrie.so
# Set environment variables
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH \
    SCROLL_PROVER_ASSETS_DIR=/snarkify-data/assets,/snarkify-data/assets \
    RUST_MIN_STACK=100000000
# Set the entrypoint for the container
ENTRYPOINT ["snarkify"]