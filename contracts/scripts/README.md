# Deployment scripts of Scroll contracts

The scripts should run as below sequence:

```bash
export layer1=l1geth # change to actual network name
export layer2=l2geth # change to actual network name

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
```

Reference testnet [run.sh](https://github.com/scroll-tech/testnet/blob/main/run.sh) for details.
