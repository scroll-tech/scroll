FROM scrolltech/cuda-go-rust-builder:cuda-11.7.1-go-1.19-rust-nightly-2022-12-10 as builder
WORKDIR /
COPY halo2-gpu /halo2-gpu
COPY scroll /scroll
RUN echo 'paths = ["/halo2-gpu/halo2_proofs"]' > /root/.cargo/config
ENV LD_LIBRARY_PATH /usr/local/cuda/lib64:$LD_LIBRARY_PATH \
    RUST_MIN_STACK 100000000 \
    CHAIN_ID 534353 \
    RUST_LOG debug
RUN echo "/scroll/roller/prover/lib" > /etc/ld.so.conf.d/a.conf && ldconfig
