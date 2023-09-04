# Download Go dependencies
FROM scrolltech/go-alpine-builder:1.19 as base

WORKDIR /src
COPY go.work* ./
COPY ./rollup/go.* ./rollup/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./prover-stats-api/go.* ./prover-stats-api/
COPY ./prover/go.* ./prover/
COPY ./tests/integration-test/go.* ./tests/integration-test/
COPY ./bridge-history-api/go.* ./bridge-history-api/
# Support mainland environment.
#ENV GOPROXY="https://goproxy.cn,direct"
RUN go mod download -x


# Build prover-stats-api
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/prover-stats-api/cmd/ && go build -v -p 4 -o /bin/prover-stats-api

# Pull prover-stats-api into a second stage deploy alpine container \
FROM alpine:latest

COPY --from=builder /bin/prover-stats-api /bin/

ENTRYPOINT ["prover-stats-api"]