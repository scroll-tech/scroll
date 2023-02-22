# Build libzkp dependency
FROM scrolltech/golang:1.18-alpine as chef
WORKDIR app

# Download Go dependencies
FROM scrolltech/golang:1.18-alpine as base
WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
COPY ./tests/integration-test/go.* ./tests/integration-test/
RUN go mod download -x


# Build coordinator
FROM base as builder
COPY . .
RUN cd ./common/libzkp && sh build.sh
RUN cd ./coordinator && go build -v -p 4 -o /bin/coordinator ./cmd


# Pull coordinator into a second stage deploy alpine container
FROM ubuntu:20.04
ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/src/coordinator/lib
ENV CHAIN_ID=534353
# ENV ZK_VERSION=
RUN mkdir -p /src/coordinator/lib
COPY --from=builder /src/common/libzkp/lib /src/coordinator/lib
COPY --from=builder /bin/coordinator /bin/


ENTRYPOINT ["/bin/coordinator"]
