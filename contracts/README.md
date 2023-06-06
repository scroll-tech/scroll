# Scroll Contracts

Note: For more comprehensive documentation, see [`./docs/`](./docs).


## Directory Structure

```
├── integration-test: Hardhat integration tests
├── lib:
│   ├── ds-test: testing tools
│   ├── forge-std: foundry dependency
│   └── solmate: testing tools
├── scripts: deployment scripts
├── src
│   ├── interfaces: common contract interfaces
│   ├── L1: contracts on the L1
│   │   ├── gateways: Gateway router and individual gateway contracts
│   │   ├── rollup: Rollup contracts for data availability and finalization
│   │   ├── IL1ScrollMessenger.sol: L1 Scroll messenger interface
│   │   └── L1ScrollMessenger.sol: L1 Scroll messenger contract
│   ├── L2: contracts on the L2
│   │   ├── gateways: Gateway router and individual gateway contracts
│   │   ├── predeploys: Predeployed contracts on the L2 messenger interface
│   │   ├── IL2ScrollMessenger.sol: L2 Scroll messenger interface
│   │   └── L2ScrollMessenger.sol: L2 Scroll messenger contract
│   ├── libraries: shared contract libraries
│   ├── mocks: mock contracts used in the testing
│   └── test: unit tests in solidity
├── foundry.toml: configure foundry
├── hardhat.config.ts: configure hardhat
├── remappings.txt: foundry dependency mappings
...
```


## Dependencies


### Foundry

First run the command below to get foundryup, the Foundry toolchain installer:

```bash
curl -L https://foundry.paradigm.xyz | bash
```

If you do not want to use the redirect, feel free to manually download the foundryup installation script from [here](https://raw.githubusercontent.com/foundry-rs/foundry/master/foundryup/foundryup).

Then, run `foundryup` in a new terminal session or after reloading your `PATH`.

Other ways to install Foundry can be found [here](https://github.com/foundry-rs/foundry#installation).


### Hardhat

```
yarn install
```


## Build

+ Run `git submodule update --init --recursive` to initialise git submodules.
+ Run `yarn prettier:solidity` to run linting in fix mode, will auto-format all solidity codes.
+ Run `yarn prettier` to run linting in fix mode, will auto-format all typescript codes.
+ Run `forge build` to compile contracts with foundry.
+ Run `npx hardhat compile` to compile with hardhat.
+ Run `forge test -vvv` to run foundry units tests. It will compile all contracts before running the unit tests.
+ Run `npx hardhat test` to run integration tests. It may not compile all contracts before running, it's better to run `npx hardhat compile` first.


## TODO

- [x] unit tests
  - [x] L1 Messenger
  - [x] L1 Gateways
  - [x] L1 Gateway Router
  - [x] L2 Messenger
  - [x] L2 Gateways
  - [x] L2 Gateway Router
  - [x] ScrollStandardERC20Factory
  - [x] Whitelist
  - [x] SimpleGasOracle
- [x] integration tests
  - [x] ERC20Gateway
  - [x] GatewayRouter
- [x] ZKRollup contracts
- [x] Gas Oracle contracts for cross chain message call
- [x] ERC721/ERC115 interface design
- [x] add proof verification codes
- [ ] security analysis
