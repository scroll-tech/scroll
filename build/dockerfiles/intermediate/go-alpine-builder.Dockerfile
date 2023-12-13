ARG GO_VERSION=1.20

FROM golang:${GO_VERSION}-alpine

# ENV GOPROXY https://goproxy.cn,direct

# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache gcc musl-dev linux-headers git ca-certificates openssl-dev
