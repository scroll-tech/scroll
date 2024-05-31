#!/bin/bash

echo ""
echo "generating config-contracts.toml"
forge script scripts/foundry/DeployScroll.s.sol:DeployScroll --sig "run(string,string)" "none" "write-config" || exit 1

echo ""
echo "generating genesis.json"
forge script scripts/foundry/DeployScroll.s.sol:GenerateGenesis || exit 1

echo ""
echo "generating rollup-config.json"
forge script scripts/foundry/DeployScroll.s.sol:GenerateRollupConfig || exit 1

echo ""
echo "generating coordinator-config.json"
forge script scripts/foundry/DeployScroll.s.sol:GenerateCoordinatorConfig || exit 1

echo ""
echo "generating chain-monitor-config.json"
forge script scripts/foundry/DeployScroll.s.sol:GenerateChainMonitorConfig || exit 1

echo ""
echo "generating bridge-history-config.json"
forge script scripts/foundry/DeployScroll.s.sol:GenerateBridgeHistoryConfig || exit 1

echo ""
echo "generating balance-checker-config.json"
forge script scripts/foundry/DeployScroll.s.sol:GenerateBalanceCheckerConfig || exit 1

echo ""
echo "generating .env.frontend"
forge script scripts/foundry/DeployScroll.s.sol:GenerateFrontendConfig || exit 1
