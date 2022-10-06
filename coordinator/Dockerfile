# Build scroll in a stock Go builder container
FROM scrolltech/go-builder:1.17 as builder

ADD ./coordinator /coordinator
ADD ./store /store

RUN cd /coordinator/cmd/coordinator/ && go build -v -p 4

# Pull scroll into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/cmd/coordinator /bin/

EXPOSE 8645
ENTRYPOINT ["coordinator"]
