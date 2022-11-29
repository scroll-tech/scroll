FROM scrolltech/l2geth:prealpha-v4.1 as GethBin

FROM alpine:latest as pre_deploy

COPY  --from=GethBin /usr/local/bin/geth ./gethbin/
RUN apk update
RUN apk add --upgrade npm
RUN apk add --upgrade git
RUN apk add --upgrade bash
RUN mkdir -p /l2geth/keystore
COPY /common/docker/l2geth/password /l2geth/
COPY /common/docker/l2geth/genesis.json /l2geth/
COPY /common/docker/l2geth/genesis-keystore /l2geth/
COPY /common/docker/l2geth/pre_run.sh /bin/
COPY /contracts /contracts
RUN sh /bin/pre_run.sh

FROM scrolltech/l2geth:prealpha-v4.1
COPY --from=pre_deploy /geth /geth
COPY --from=pre_deploy /contracts/deployments /deployments
COPY /common/docker/l2geth/password password
COPY /common/docker/l2geth/genesis.json genesis.json
RUN  mkdir -p /keystore
COPY --from=pre_deploy /keystore/genesis-keystore /keystore/
COPY /common/docker/l2geth/l2geth_run.sh /bin/

ENTRYPOINT ["sh", "/bin/l2geth_run.sh"]
