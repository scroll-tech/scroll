# Deployment scripts of Scroll contracts

## Deployment using Hardhat

The scripts should run as below sequence:

```bash
export layer1=l1geth # change to actual network name
export layer2=l2geth # change to actual network name
export owner=0x0000000000000000000000000000000000000000 # change to actual owner

# deploy contracts in layer 1
npx hardhat --network $layer1 run scripts/deploy_proxy_admin.ts
npx hardhat --network $layer1 run scripts/deploy_zkrollup.ts
env CONTRACT_NAME=L1ScrollMessenger npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1GatewayRouter npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1StandardERC20Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1CustomERC20Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1ERC721Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1ERC1155Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts

# deploy contracts in layer 2, note: l2_messenger is predeployed
npx hardhat --network $layer2 run scripts/deploy_proxy_admin.ts
npx hardhat --network $layer2 run scripts/deploy_l2_messenger.ts
npx hardhat --network $layer2 run scripts/deploy_l2_token_factory.ts
env CONTRACT_NAME=L2GatewayRouter npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2StandardERC20Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2CustomERC20Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2ERC721Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2ERC1155Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts

# initalize contracts in layer 1, should set proper bash env variables first
npx hardhat --network $layer1 run scripts/initialize_l1_erc20_gateway.ts
npx hardhat --network $layer1 run scripts/initialize_l1_gateway_router.ts
npx hardhat --network $layer1 run scripts/initialize_zkrollup.ts
npx hardhat --network $layer1 run scripts/initialize_l1_messenger.ts
npx hardhat --network $layer1 run scripts/initialize_l1_custom_erc20_gateway.ts
npx hardhat --network $layer1 run scripts/initialize_l1_erc1155_gateway.ts
npx hardhat --network $layer1 run scripts/initialize_l1_erc721_gateway.ts

# initalize contracts in layer 2, should set proper bash env variables first
npx hardhat --network $layer2 run scripts/initialize_l2_erc20_gateway.ts
npx hardhat --network $layer2 run scripts/initialize_l2_gateway_router.ts
npx hardhat --network $layer2 run scripts/initialize_l2_custom_erc20_gateway.ts
npx hardhat --network $layer2 run scripts/initialize_l2_erc1155_gateway.ts
npx hardhat --network $layer2 run scripts/initialize_l2_erc721_gateway.ts
npx hardhat --network $layer2 run scripts/initialize_l2_token_factory.ts

# transfer ownership in layer 1
env CONTRACT_NAME=ProxyAdmin CONTRACT_OWNER=$owner npx hardhat run --network $layer1 scripts/transfer_ownership.ts
env CONTRACT_NAME=L1ScrollMessenger CONTRACT_OWNER=$owner npx hardhat run --network $layer1 scripts/transfer_ownership.ts
env CONTRACT_NAME=ZKRollup CONTRACT_OWNER=$owner npx hardhat run --network $layer1 scripts/transfer_ownership.ts
env CONTRACT_NAME=L1GatewayRouter CONTRACT_OWNER=$owner npx hardhat run --network $layer1 scripts/transfer_ownership.ts
env CONTRACT_NAME=L1CustomERC20Gateway CONTRACT_OWNER=$owner npx hardhat run --network $layer1 scripts/transfer_ownership.ts
env CONTRACT_NAME=L1ERC721Gateway CONTRACT_OWNER=$owner npx hardhat run --network $layer1 scripts/transfer_ownership.ts
env CONTRACT_NAME=L1ERC1155Gateway CONTRACT_OWNER=$owner npx hardhat run --network $layer1 scripts/transfer_ownership.ts
# transfer ownership in layer 2
env CONTRACT_NAME=ProxyAdmin CONTRACT_OWNER=$owner npx hardhat run --network $layer2 scripts/transfer_ownership.ts
env CONTRACT_NAME=L2ScrollMessenger CONTRACT_OWNER=$owner npx hardhat run --network $layer2 scripts/transfer_ownership.ts
env CONTRACT_NAME=L2GatewayRouter CONTRACT_OWNER=$owner npx hardhat run --network $layer2 scripts/transfer_ownership.ts
env CONTRACT_NAME=L2CustomERC20Gateway CONTRACT_OWNER=$owner npx hardhat run --network $layer2 scripts/transfer_ownership.ts
env CONTRACT_NAME=L2ERC721Gateway CONTRACT_OWNER=$owner npx hardhat run --network $layer2 scripts/transfer_ownership.ts
env CONTRACT_NAME=L2ERC1155Gateway CONTRACT_OWNER=$owner npx hardhat run --network $layer2 scripts/transfer_ownership.ts
```

Reference testnet [run_deploy_contracts.sh](https://github.com/scroll-tech/testnet/blob/staging/run_deploy_contracts.sh) for details.

## Deployment using Foundry

Note: The Foundry scripts take parameters like `CHAIN_ID_L2` and `L1_ZK_ROLLUP_PROXY_ADDR` as environment variables.

```bash
# allexport
$ set -a

$ cat .env
CHAIN_ID_L2="5343541"
SCROLL_L1_RPC="http://localhost:8543"
SCROLL_L2_RPC="http://localhost:8545"
L1_DEPLOYER_PRIVATE_KEY="0x0000000000000000000000000000000000000000000000000000000000000001"
L2_DEPLOYER_PRIVATE_KEY="0x0000000000000000000000000000000000000000000000000000000000000002"
L1_ROLLUP_OPERATOR_ADDR="0x1111111111111111111111111111111111111111"

$ source .env

# Deploy L1 contracts
# Note: We extract the logged addresses as environment variables.
$ OUTPUT=$(forge script scripts/foundry/DeployL1BridgeContracts.s.sol:DeployL1BridgeContracts --rpc-url $SCROLL_L1_RPC --broadcast); echo $OUTPUT
$ echo "$OUTPUT" | grep -Eo "(L1)_.*" > .env.l1_addresses
$ source .env.l1_addresses

# Deploy L2 contracts
$ OUTPUT=$(forge script scripts/foundry/DeployL2BridgeContracts.s.sol:DeployL2BridgeContracts --rpc-url $SCROLL_L2_RPC --broadcast); echo $OUTPUT
$ echo "$OUTPUT" | grep -Eo "(L2)_.*" > .env.l2_addresses
$ source .env.l2_addresses

# Initialize contracts
$ forge script scripts/foundry/InitializeL1BridgeContracts.s.sol:InitializeL1BridgeContracts --rpc-url $SCROLL_L1_RPC --broadcast
$ forge script scripts/foundry/InitializeL2BridgeContracts.s.sol:InitializeL2BridgeContracts --rpc-url $SCROLL_L2_RPC --broadcast
```