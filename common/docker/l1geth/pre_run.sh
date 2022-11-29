#!/bin/sh

if [ ! -f ./keystore ]; then
  echo "initializing l1geth"
  cp /l1geth/genesis.json /l1geth/password ./
  ./gethbin/geth --datadir . init genesis.json
  cp /l1geth/genesis-keystore ./keystore/
fi

nohup ./gethbin/geth --mine --datadir "." --unlock 0 --password "./password" --allow-insecure-unlock --nodiscover \
  --http --ws --ws.addr "127.0.0.1" --ws.port 8546 > l1geth.log 2>&1 &

# this function deploys mock contracts in the scroll_l2 image
deploy_bridge_contracts() {
    cd contracts
    export SCROLL_L1_RPC="http://127.0.0.1:8545"
    export L1_DEPLOYER_PRIVATE_KEY="1212121212121212121212121212121212121212121212121212121212121212"
    export CHAIN_ID_L1="52077"
    git init
    npm install @openzeppelin/contracts
    git submodule add https://github.com/dapphub/ds-test
    mv ds-test lib/
    git submodule add https://github.com/foundry-rs/forge-std
    mv forge-std lib/
    git submodule add https://github.com/rari-capital/solmate
    mv solmate lib/

    export layer1=l1geth # change to actual network name
    export owner=0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63 # change to actual owner

    # deploy contracts in layer 1
    npx hardhat --network $layer1 run scripts/deploy_proxy_admin.ts
    npx hardhat --network $layer1 run scripts/deploy_zkrollup.ts
    env CONTRACT_NAME=L1ScrollMessenger npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L1GatewayRouter npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L1StandardERC20Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L1CustomERC20Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L1ERC721Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L1ERC1155Gateway npx hardhat run --network $layer1 scripts/deploy_proxy_contract.ts

    # initalize contracts in layer 1, should set proper bash env variables first
    #npx hardhat --network $layer1 run scripts/initialize_l1_erc20_gateway.ts
    #npx hardhat --network $layer1 run scripts/initialize_l1_gateway_router.ts
    #npx hardhat --network $layer1 run scripts/initialize_zkrollup.ts
    #npx hardhat --network $layer1 run scripts/initialize_l1_messenger.ts
    #npx hardhat --network $layer1 run scripts/initialize_l1_custom_erc20_gateway.ts
    #npx hardhat --network $layer1 run scripts/initialize_l1_erc1155_gateway.ts
    #npx hardhat --network $layer1 run scripts/initialize_l1_erc721_gateway.ts

}

deploy_bridge_contracts
