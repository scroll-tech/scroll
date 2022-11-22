FROM scrolltech/l2geth:prealpha-v4.1 as GethBin


#FROM rust:alpine3.16
FROM ghcr.io/foundry-rs/foundry:latest

COPY  --from=GethBin /usr/local/bin/geth ./gethbin/
RUN apk update
RUN apk add --upgrade npm
RUN apk add --upgrade git
RUN apk add --upgrade bash
RUN mkdir -p /l2geth/keystore
COPY /common/docker/l2geth/password /l2geth/
COPY /common/docker/l2geth/genesis.json /l2geth/
COPY /common/docker/l2geth/genesis-keystore /l2geth/
COPY /common/docker/l2geth/l2geth_run.sh /bin/
COPY /common/docker/l2geth/pre_run.sh /bin/
COPY /contracts /contracts
RUN sh /bin/pre_run.sh

ENTRYPOINT ["sh", "/bin/l2geth_run.sh"]
