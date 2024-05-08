FROM ubuntu:20.04

ARG GO_VERSION=1.21.1
RUN apt-get update
RUN apt-get install -y wget
RUN apt-get install -y gcc
RUN apt-get install -y make
RUN apt-get install -y docker.io
RUN apt-get install -y docker-compose
RUN rm -rf /usr/local/go
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
RUN rm go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
RUN wget https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libzktrie.so && mv libzktrie.so /usr/local/lib
RUN wget https://github.com/scroll-tech/da-codec/releases/download/v0.0.0-rc0-ubuntu20.04/libscroll_zstd.so && mv libscroll_zstd.so /usr/local/lib
ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/local/lib
ENV CGO_ENABLED=1
