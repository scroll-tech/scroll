# Scroll Contracts

Note: For more comprehensive documentation, see [`./docs/`](./docs).

## Directory Structure

```
integration-test
|- xxx.test.ts - "Hardhat integration tests"
lib
|- forge-std - "foundry dependency"
scripts
|- deploy_xxx.ts - "hardhat deploy script"
|- foundry - "foundry deploy scripts"
src
|- test
|  `- xxx.t.sol - "Unit testi in solidity"
`- xxx.sol - "solidity contract"
.gitmodules -  "foundry dependecy modules"
foundry.toml - "configure foundry"
hardhat.config.ts - "configure hardhat"
remappings.txt - "foundry dependency mappings"
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

- Run `git submodule update --init --recursive` to initialise git submodules.
- Run `yarn prettier:solidity` to run linting in fix mode, will auto-format all solidity codes.
- Run `yarn prettier` to run linting in fix mode, will auto-format all typescript codes.
- Run `yarn prepare` to install the precommit linting hook
- Run `forge build` to compile contracts with foundry.
- Run `npx hardhat compile` to compile with hardhat.
- Run `forge test -vvv` to run foundry units tests. It will compile all contracts before running the unit tests.
- Run `npx hardhat test` to run integration tests. It may not compile all contracts before running, it's better to run `npx hardhat compile` first.

## TODO

- [ ] unit tests
  - [ ] L1 Messenger
  - [x] L1 Gateways
  - [x] L1 Gateway Router
  - [ ] L2 Messenger
  - [x] L2 Gateways
  - [x] L2 Gateway Router
  - [x] ScrollStandardERC20Factory
  - [x] Whitelist
  - [ ] SimpleGasOracle
- [ ] integration tests
  - [x] ERC20Gateway
  - [x] GatewayRouter
- [ ] ZKRollup contracts
- [x] Gas Oracle contracts for cross chain message call
- [ ] ERC721/ERC115 interface design
- [ ] add proof verification codes
- [ ] security analysis
