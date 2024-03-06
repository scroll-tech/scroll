FROM ethereum/client-go:v1.13.4

COPY password /l1geth/
COPY genesis.json /l1geth/
COPY genesis-keystore /l1geth/
COPY l1geth_run.sh /bin/

ENTRYPOINT ["sh", "/bin/l1geth_run.sh"]