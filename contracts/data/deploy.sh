#!/bin/sh

export FOUNDRY_EVM_VERSION="cancun"
export FOUNDRY_BYTECODE_HASH="none"

# deploy L1
echo ""
echo "deploying on L1"
forge script scripts/foundry/DeployScroll.s.sol:DeployScroll --rpc-url "$L1_RPC_ENDPOINT" --sig "run(string,string)" "L1" "none" --broadcast

# deploy L2
echo ""
echo "deploying on L2"
forge script scripts/foundry/DeployScroll.s.sol:DeployScroll --rpc-url "$L2_RPC_ENDPOINT" --sig "run(string,string)" "L2" "none" --broadcast
