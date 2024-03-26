FROM scrolltech/l2geth:scroll-v5.1.26

RUN mkdir -p /l2geth/keystore

COPY password /l2geth/
COPY genesis.json /l2geth/
COPY genesis-keystore /l2geth/
COPY l2geth_run.sh /bin/

ENTRYPOINT ["sh", "/bin/l2geth_run.sh"]
