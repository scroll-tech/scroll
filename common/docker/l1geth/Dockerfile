FROM ethereum/client-go as gethbin

FROM alpine:latest
COPY --from=gethbin /usr/local/bin/geth ./gethbin/
RUN apk update
RUN apk add --upgrade npm
RUN apk add --upgrade git
RUN apk add --upgrade bash
RUN npm install -g hardhat
COPY /contracts /contracts
COPY /common/docker/l1geth/password /l1geth/
COPY /common/docker/l1geth/genesis.json /l1geth/
COPY /common/docker/l1geth/genesis-keystore /l1geth/
COPY /common/docker/l1geth/pre_run.sh /bin/
RUN sh /bin/pre_run.sh
COPY /common/docker/l1geth/l1geth_run.sh /bin/

ENTRYPOINT ["sh", "/bin/l1geth_run.sh"]