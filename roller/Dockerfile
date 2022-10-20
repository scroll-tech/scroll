# Build roller in a stock Go builder container
FROM golang:1.18-alpine as builder

ENV GOPROXY https://goproxy.io,direct

COPY ./ /
RUN apk add --no-cache gcc && cd /roller/cmd/ && go build -v -p 4 -o roller

# Pull roller into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /roller/cmd/ /bin/

ENTRYPOINT ["roller"]
