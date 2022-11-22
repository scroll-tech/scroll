#!/bin/sh

if [ ! -f ./keystore ]; then
  echo "initializing l1geth"
  cp /l1geth/genesis.json /l1geth/password ./
  geth --datadir . init genesis.json
  cp /l1geth/genesis-keystore ./keystore/
fi