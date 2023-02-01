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

# Build bridge
FROM base as builder

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/bridge/cmd && go build -v -p 4 -o /bin/bridge

# Pull bridge into a second stage deploy alpine container
FROM ubuntu:latest

COPY --from=builder /bin/bridge /bin/
RUN  apt-get update -y \
&& apt-get install iputils-ping netcat net-tools -y \
&& apt-get clean \
&& rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["bridge"]
