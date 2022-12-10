# Build scroll in a stock Go builder container
FROM scrolltech/full-builder:go-1.17-rust-nightly-2022-08-23 as zkp-builder

COPY ./ /src/

RUN cd /src/common/libzkp/impl && cargo build --release && cp ./target/release/libzkp.a ../interface/
RUN cp -r /src/common/libzkp/interface /src/coordinator/verifier/lib


# Download Go dependencies
FROM scrolltech/full-builder:go-1.18-rust-nightly-2022-08-23 as base

WORKDIR /src
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

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /coordinator && go build -v -p 4 -o /bin/coordinator ./cmd

# Pull coordinator into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/coordinator /bin/

ENTRYPOINT ["/bin/coordinator"]

