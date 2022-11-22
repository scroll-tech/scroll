FROM ethereum/client-go

COPY password /l1geth/
COPY genesis.json /l1geth/
COPY genesis-keystore /l1geth/
COPY pre_run.sh /bin/
RUN sh /bin/pre_run.sh
COPY l1geth_run.sh /bin/

ENTRYPOINT ["sh", "/bin/l1geth_run.sh"]