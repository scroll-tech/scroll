#!/bin/sh

if [ ! -f ./keystore ]; then
  echo "initializing l2geth"
  cp /l2geth/genesis.json /l2geth/password ./
  ./gethbin/geth --datadir . init genesis.json
  cp /l2geth/genesis-keystore ./keystore/
fi

nohup ./gethbin/geth --mine --datadir "." --unlock 0 --password "./password" --allow-insecure-unlock --gcmode archive --verbosity 3 \
  --http --http.addr "127.0.0.1" --http.port 8545 --http.api "eth,scroll,net,web3,debug" \
  --ws --ws.addr "127.0.0.1" --ws.port 8546 --ws.api "eth,scroll,net,web3,debug" > l2geth.log 2>&1 &

# this function deploys mock contracts in the scroll_l2 image
deploy_bridge_contracts() {
    cd contracts
    export SCROLL_L2_RPC="http://127.0.0.1:8545"
    export L2_DEPLOYER_PRIVATE_KEY="1212121212121212121212121212121212121212121212121212121212121212"
    export CHAIN_ID_L2="53077"
    git init
    npm install @openzeppelin/contracts
    git submodule add https://github.com/dapphub/ds-test
    mv ds-test lib/
    git submodule add https://github.com/foundry-rs/forge-std
    mv forge-std lib/
    git submodule add https://github.com/rari-capital/solmate
    mv solmate lib/

    export layer2=l2geth # change to actual network name
    export owner=0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63 # change to actual owner
    npx hardhat --network $layer2 run scripts/deploy_proxy_admin.ts
    npx hardhat --network $layer2 run scripts/deploy_l2_messenger.ts
    npx hardhat --network $layer2 run scripts/deploy_l2_token_factory.ts
    env CONTRACT_NAME=L2GatewayRouter npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L2StandardERC20Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L2CustomERC20Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L2ERC721Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts
    env CONTRACT_NAME=L2ERC1155Gateway npx hardhat run --network $layer2 scripts/deploy_proxy_contract.ts

    # initialize these need to run seperately, maybe not in docker build stage
    #npx hardhat --network $layer2 run scripts/initialize_l2_erc20_gateway.ts
    #npx hardhat --network $layer2 run scripts/initialize_l2_gateway_router.ts
    #npx hardhat --network $layer2 run scripts/initialize_l2_custom_erc20_gateway.ts
    #npx hardhat --network $layer2 run scripts/initialize_l2_erc1155_gateway.ts
    #npx hardhat --network $layer2 run scripts/initialize_l2_erc721_gateway.ts
    #npx hardhat --network $layer2 run scripts/initialize_l2_token_factory.ts
}

deploy_bridge_contracts