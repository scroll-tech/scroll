# Build scroll in a stock Go builder container
FROM scrolltech/go-rust-builder:go-1.17-rust-nightly-2022-08-23 as zkp-builder

COPY ./ /

RUN cd /common/zkp/rust && cargo build --release && cp ./target/release/libzkp.a ../lib/
RUN cp -r /common/zkp/lib /coordinator/verifier/

FROM scrolltech/go-builder:1.18 as builder

COPY --from=zkp-builder / /

RUN cd /coordinator && go build -v -p 4 -o coordinator ./cmd

# Pull scroll into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/coordinator /bin/

ENTRYPOINT ["coordinator"]
