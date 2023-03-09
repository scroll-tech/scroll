#!/bin/sh

if [ ! -f ./keystore ]; then
  echo "initializing l1geth"
  cp /l1geth/genesis.json /l1geth/password ./
  geth --datadir . init genesis.json
  cp /l1geth/genesis-keystore ./keystore/
fi

if [ ! -n "${IPC_PATH}" ];then
  IPC_PATH="/tmp/l1geth_path.ipc"
fi

exec geth --mine --datadir "." --unlock 0 --password "./password" --allow-insecure-unlock --nodiscover \
  --http --http.addr "0.0.0.0" --http.port 8545 --ws --ws.addr "0.0.0.0" --ws.port 8546 --ipcpath ${IPC_PATH}

