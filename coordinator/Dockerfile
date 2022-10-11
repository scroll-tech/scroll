# Build scroll in a stock Go builder container
FROM scrolltech/go-builder:1.17 as builder

COPY ./coordinator /coordinator
COPY ./store /store
COPY ./utils /utils

RUN cd /coordinator/cmd/ && go build -v -p 4

# Pull scroll into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /coordinator/cmd /bin/

EXPOSE 8645
ENTRYPOINT ["coordinator"]
