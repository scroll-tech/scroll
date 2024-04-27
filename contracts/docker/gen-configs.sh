#!/bin/bash

# generate contract addresses
echo ""
echo "generating config-contracts.toml"
forge script scripts/foundry/DeployScroll.s.sol:DeployScroll --sig "run(string,string)" "none" "write-config" || exit 1

# generate genesis
echo ""
echo "generating genesis.json"
forge script scripts/foundry/DeployScroll.s.sol:GenerateGenesis || exit 1
