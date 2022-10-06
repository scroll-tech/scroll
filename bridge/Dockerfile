# Build bridge in a stock Go builder container
FROM scrolltech/go-builder:1.17 as builder

ADD ./bridge /bridge
ADD ./store /store

RUN cd /bridge/cmd/bridge/ && go build -v -p 4

# Pull bridge into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bridge/cmd/bridge /bin/

EXPOSE 8645
ENTRYPOINT ["bridge"]
