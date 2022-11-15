# Build scroll in a stock Go builder container
FROM scrolltech/go-rust-builder:go-1.17-rust-nightly-2022-08-23 as builder

COPY ./ /

RUN cd ../common/zkp/rust && cargo build --release && cp ./target/release/libzkp.a ../lib/
RUN cp -r ../common/zkp/lib ./verifier/

RUN cd /coordinator/cmd && go build -v -p 4 -o coordinator

# Pull scroll into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/cmd/coordinator /bin/

ENTRYPOINT ["coordinator"]
