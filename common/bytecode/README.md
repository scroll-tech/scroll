## How to pre deploy contracts?
* Please reference to https://github.com/scroll-tech/genesis-creator.
1. Setup env
```bash
   git clone git@github.com:scroll-tech/genesis-creator.git
   cd genesis-creator
   go get -v github.com/scroll-tech/go-ethereum@develop && go mod tidy
   make abi && make genesis-creator
   make l2geth-docker
```

2. Start docker and write pre deployed contracts into genesis file.
```bash
   make start-docker
   ./bin/genesis-creator -genesis ${SCROLLPATH}/common/docker/l2geth/genesis.json -contract [erc20|greeter]
```

3. Rebuild l2geth docker.
```bash
   cd ${SCROLLPATH}
   make dev_docker
```

## How to get contract abi?
* Other contracts' step same to eth20, e.g:
1. Install solc.
   
    *Reference to https://docs.soliditylang.org/en/latest/installing-solidity.html*

2. Get abi file.
```bash
   cd genesis-creator
   solc --combined-json "abi" --optimize ${SCROLLPATH}/common/bytecode/erc20/ERC20Mock.sol | jq > ${SCROLLPATH}/common/bytecode/erc20/ERC20Mock.json
```

3. Translate abi to go.
```bash
   cd ${SCROLLPATH}
   make -C common/bytecode all
```
