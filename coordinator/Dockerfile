# Build scroll in a stock Go builder container
FROM scrolltech/go-builder:1.18 as builder

COPY ./ /

RUN cd /coordinator/cmd && go build -v -p 4 -o coordinator

# Pull scroll into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/cmd/coordinator /bin/

ENTRYPOINT ["coordinator"]
