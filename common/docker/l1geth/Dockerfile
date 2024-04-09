FROM ethereum/client-go:stable

COPY password /l1geth/
COPY genesis.json /l1geth/
COPY genesis-keystore /l1geth/
COPY l1geth_run.sh /bin/

# Expose ports used by the Ethereum node
EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["sh", "/bin/l1geth_run.sh"]