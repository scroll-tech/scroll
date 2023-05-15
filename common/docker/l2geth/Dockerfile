FROM scrolltech/l2geth:scroll-v3.1.12

RUN mkdir -p /l2geth/keystore

COPY password /l2geth/
COPY genesis.json /l2geth/
COPY genesis-keystore /l2geth/
COPY l2geth_run.sh /bin/

ENTRYPOINT ["sh", "/bin/l2geth_run.sh"]
