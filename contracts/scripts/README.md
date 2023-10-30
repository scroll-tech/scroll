# Deployment scripts of Scroll contracts

## Deployment using Hardhat

The scripts should run as below sequence:
complete .env file
$ source .env

# deploy contracts in layer 1

npx hardhat --network $layer1 run scripts/deploy_proxy_admin.ts
npx hardhat --network $layer1 run scripts/deploy_whitelist.ts
env CONTRACT_NAME=L1MessageQueue npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2GasPriceOracle npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
npx hardhat --network $layer1 run scripts/deploy_scroll_chain.ts
env CONTRACT_NAME=L1ScrollMessenger npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1GatewayRouter npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1StandardERC20Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1CustomERC20Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1ERC721Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1ERC1155Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L1ETHGateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=EnforcedTxGateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2TxFeeVault npx hardhat run --network $layer1 scripts/deploy_predeploys.ts
#env CONTRACT_NAME=L1WETHGateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts

# deploy contracts in layer 2, note: l2_messenger is predeployed

env CONTRACT_NAME=L1GasPriceOracle npx hardhat run --network $layer2 scripts/deploy_predeploys.ts
env CONTRACT_NAME=L2MessageQueue npx hardhat run --network $layer2 scripts/deploy_predeploys.ts
env CONTRACT_NAME=L2TxFeeVault npx hardhat run --network $layer2 scripts/deploy_predeploys.ts
env CONTRACT_NAME=L1BlockContainer npx hardhat run --network $layer2 scripts/deploy_predeploys.ts #部署 L1BlockContainer 后将地址添加进.env
npx hardhat --network $layer2 run scripts/deploy_whitelist.ts
npx hardhat --network $layer2 run scripts/deploy_proxy_admin.ts
npx hardhat --network $layer2 run scripts/deploy_l2_messenger.ts
npx hardhat --network $layer2 run scripts/deploy_l2_token_factory.ts
env CONTRACT_NAME=L2GatewayRouter npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2StandardERC20Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2CustomERC20Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2ERC721Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2ERC1155Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
env CONTRACT_NAME=L2ETHGateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
#env CONTRACT_NAME=L2WETHGateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts

# initalize contracts in layer 1, should set proper bash env variables first

npx hardhat --network $layer1 run scripts/initialize_scroll_chain.ts
npx hardhat --network $layer1 run scripts/initialize_l2_gasprice_oracle.ts
npx hardhat --network $layer1 run scripts/initialize_l1_message_queue.ts
npx hardhat --network $layer1 run scripts/initialize_l1_messenger.ts
npx hardhat --network $layer1 run scripts/initialize_l1_eth_gateway.ts

# npx hardhat --network $layer1 run scripts/initialize_l1_erc20_gateway.ts

# npx hardhat --network $layer1 run scripts/initialize_l1_gateway_router.ts

# npx hardhat --network $layer1 run scripts/initialize_l1_messenger.ts

# npx hardhat --network $layer1 run scripts/initialize_l1_custom_erc20_gateway.ts

# npx hardhat --network $layer1 run scripts/initialize_l1_erc1155_gateway.ts

# npx hardhat --network $layer1 run scripts/initialize_l1_erc721_gateway.ts

# initalize contracts in layer 2, should set proper bash env variables first

npx hardhat --network $layer2 run scripts/initialize_l2_predeploys.ts
npx hardhat --network $layer2 run scripts/initialize_l2_scroll_messenger.ts
npx hardhat --network $layer2 run scripts/initialize_l2_eth_gateway.ts

# npx hardhat --network $layer2 run scripts/initialize_l2_erc20_gateway.ts

# npx hardhat --network $layer2 run scripts/initialize_l2_gateway_router.ts

# npx hardhat --network $layer2 run scripts/initialize_l2_custom_erc20_gateway.ts

# npx hardhat --network $layer2 run scripts/initialize_l2_erc1155_gateway.ts

# npx hardhat --network $layer2 run scripts/initialize_l2_erc721_gateway.ts

# npx hardhat --network $layer2 run scripts/initialize_l2_token_factory.ts

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

```
