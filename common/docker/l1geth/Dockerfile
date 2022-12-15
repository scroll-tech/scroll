FROM ethereum/client-go as gethbin

FROM alpine:latest as pre_deploy
COPY --from=gethbin /usr/local/bin/geth ./gethbin/
RUN apk update
RUN apk add --upgrade npm git bash
RUN npm install -g hardhat
COPY /contracts /contracts
COPY /common/docker/l1geth/password /l1geth/
COPY /common/docker/l1geth/genesis.json /l1geth/
COPY /common/docker/l1geth/genesis-keystore /l1geth/
COPY /common/docker/l1geth/delpoy_contracts.sh /bin/
RUN sh /bin/delpoy_contracts.sh


FROM ethereum/client-go
COPY --from=pre_deploy /geth /geth
COPY --from=pre_deploy /contracts/deployments /deployments
COPY /common/docker/l1geth/password password
COPY /common/docker/l1geth/genesis.json genesis.json
RUN  mkdir -p /keystore
COPY --from=pre_deploy /keystore/genesis-keystore /keystore/
COPY /common/docker/l1geth/l1geth_run.sh /bin/

ENTRYPOINT ["sh", "/bin/l1geth_run.sh"]