# Build scroll in a stock Go builder container
FROM scrolltech/go-rust-builder:go-1.17-rust-nightly-2022-08-23 as zkp-builder

COPY ./ /src/

RUN cd /src/common/zkp/rust && cargo build --release && cp ./target/release/libzkp.a ../lib/
RUN cp -r /src/common/zkp/lib /src/coordinator/verifier/


# Download Go dependencies
FROM scrolltech/go-builder:1.18 as base

COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
RUN go mod download -x


# Build coordinator
FROM base as builder

COPY --from=zkp-builder /src/ /

RUN cd /coordinator && go build -v -p 4 -o coordinator ./cmd


# Pull coordinator into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/coordinator /bin/

ENTRYPOINT ["coordinator"]
