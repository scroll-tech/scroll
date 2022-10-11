# Build bridge in a stock Go builder container
FROM scrolltech/go-builder:1.17 as builder

COPY ./bridge /bridge
COPY ./store /store
COPY ./utils /utils

RUN cd /bridge/cmd/ && go build -v -p 4

# Pull bridge into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /bridge/cmd /bin/

EXPOSE 8645
ENTRYPOINT ["bridge"]
