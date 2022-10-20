# Build roller in a stock Go builder container
FROM golang:1.18-alpine as builder

ENV GOPROXY https://goproxy.io,direct

COPY go.work go.work
COPY go.work.sum go.work.sum
COPY ./roller /go-roller
COPY ./common ./common
RUN cd /go-roller/cmd/ && go build -v -p 4 -o roller

# Pull roller into a second stage deploy alpine container
FROM alpine:latest

COPY --from=builder /go-roller/cmd/ /bin/

ENTRYPOINT ["roller"]
