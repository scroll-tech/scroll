#!/bin/bash
set -ue

# This script is used to generate the typechain artifacts for the contracts

mkdir -p abis types
cat ../artifacts/src/Safe.sol/Safe.json | jq .abi >> abis/safe.json
cat ../artifacts/src/TimelockController.sol/TimelockController.json | jq .abi >> abis/timelock.json
cat ../artifacts/src/Forwarder.sol/Forwarder.json | jq .abi >> abis/forwarder.json

npx typechain --target=ethers-v6 "abis/*.json" 