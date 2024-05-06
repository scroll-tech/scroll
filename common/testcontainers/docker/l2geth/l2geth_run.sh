#!/bin/sh

if [ ! -f ./keystore ]; then
  echo "initializing l2geth"
  cp /l2geth/genesis.json /l2geth/password ./
  geth --datadir . init genesis.json
  cp /l2geth/genesis-keystore ./keystore/
fi

if [ ! -n "${IPC_PATH}" ];then
  IPC_PATH="/tmp/l2geth_path.ipc"
fi

exec geth --mine --datadir "." --unlock 0 --password "./password" --allow-insecure-unlock --gcmode archive --verbosity 3 \
  --http --http.addr "0.0.0.0" --http.port 8545 --http.api "eth,scroll,net,web3,debug" \
  --ws --ws.addr "0.0.0.0" --ws.port 8546 --ws.api "eth,scroll,net,web3,debug" \
  --ccc \
  --ipcpath ${IPC_PATH}
