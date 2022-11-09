# Build scroll in a stock Go builder container
FROM scrolltech/go-rust-builder:go-1.18-rust-nightly-2022-08-23 as builder

COPY ./ /

RUN apt install -y make

RUN cd /coordinator && make coordinator

# Pull scroll into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/cmd/coordinator /bin/

ENTRYPOINT ["coordinator"]
