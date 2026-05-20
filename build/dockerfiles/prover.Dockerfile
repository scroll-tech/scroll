# Scroll zkVM Prover — Production-style packaging
# Binary is built externally (CI or local) using the cuda-go-rust-builder
# and copied into a minimal CUDA runtime image.
ARG RUNTIME_IMAGE=nvidia/cuda:12.9.1-runtime-ubuntu22.04

FROM ${RUNTIME_IMAGE}
WORKDIR /prover

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends libssl-dev curl ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Install solc (needed by prover for EVM proof generation)
RUN curl -sL https://github.com/ethereum/solidity/releases/download/v0.8.24/solc-static-linux -o /usr/local/bin/solc && \
    chmod +x /usr/local/bin/solc

# Copy pre-built prover binary from host (built via `make prover` with CUDA)
COPY target/release/prover /usr/local/bin/

ENV LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH

ENTRYPOINT ["prover"]
