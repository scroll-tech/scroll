# Download Go dependencies
FROM scrolltech/go-alpine-builder:1.18 as base

WORKDIR /src
COPY go.work* ./
COPY ./bridge/go.* ./bridge/
COPY ./common/go.* ./common/
COPY ./coordinator/go.* ./coordinator/
COPY ./database/go.* ./database/
COPY ./roller/go.* ./roller/
COPY ./tests/integration-test/go.* ./tests/integration-test/
RUN go mod download -x

# Build gas_oracle
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge/cmd/gas_oracle/ && go build -v -p 4 -o /bin/gas_oracle

# Pull gas_oracle into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bin/gas_oracle /bin/

ENTRYPOINT ["gas_oracle"]