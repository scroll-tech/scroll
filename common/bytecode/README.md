## How to pre deploy contracts?
* Please reference to https://github.com/scroll-tech/genesis-creator.
```
1. Setup env
    git clone git@github.com:scroll-tech/genesis-creator.git
    cd genesis-creator
    go get -v github.com/scroll-tech/go-ethereum@staging && go mod tidy
    make abi && make genesis-creator
    make l2geth-docker

2. Start docker and get pre deployed contracts
    make start-docker
    ./bin/genesis-creator -genesis ${SCROLLPATH}/common/docker/l2geth/genesis.json -contract [dao|erc20|greeter|nft|scroll.l1|scroll.l2|sushi|uniswap|vote]

3. Rebuild l2geth docker in local env
    cd ${SCROLLPATH}
    make dev_docker
```

## How to get contracts' abi?
* Other contracts' step same to eth20, i.g:
```
1. Download openzeppelin and set env
    git clone git@github.com:OpenZeppelin/openzeppelin-contracts.git
    cd openzeppelin-contracts
    yarn install
    truffle init -y
    truffle compile
    npm install truffle-flattener -g
    truffle-flattener contracts/mocks/ERC20Mock.sol > ${SCROLLPATH}/common/bytecode/erc20/erc20.sol

2. Install solc in local machine
    Reference to https://docs.soliditylang.org/en/latest/installing-solidity.html

3. Get abi type of json file
    solc --combined-json "abi" --optimize ${SCROLLPATH}/common/bytecode/erc20/ERC20Mock.sol | jq > ${SCROLLPATH}/common/bytecode/erc20/ERC20Mock.json
    
4. Translate abi to go
    cd ${SCROLLPATH}
    make -C common/bytecode all
```