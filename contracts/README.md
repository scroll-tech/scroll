# Scroll Contracts

This directory contains the solidity code for Scroll L1 bridge and rollup contracts and L2 bridge and pre-deployed contracts. The [`specs`](../specs/) folder describes the overall Scroll protocol including the cross-domain messaging and rollup process. You can also find contract APIs and more details in the [`docs`](./docs) folder.

## Directory Structure

<pre>
├── <a href="./docs/">docs</a>: Documentation for the contracts
├── <a href="./integration-test/">integration-test</a>: Hardhat integration tests
├── <a href="./lib/">lib</a>
│   ├── <a href="./lib/ds-test">ds-test</a>: testing tools
│   ├── <a href="./lib/forge-std">forge-std</a>: foundry dependency
│   └── <a href="./lib/solmate">solmate</a>: testing tools
├── <a href="./scripts">scripts</a>: deployment scripts
├── <a href="./src">src</a>
│   ├── <a href="./src/interfaces/">interfaces</a>: common contract interfaces
│   ├── <a href="./src/L1/">L1</a>: contracts on the L1
│   │   ├── <a href="./src/L1/gateways/">gateways</a>: Gateway router and individual gateway contracts
│   │   ├── <a href="./src/L1/rollup/">rollup</a>: Rollup contracts for data availability and finalization
│   │   ├── <a href="./src/L1/IL1ScrollMessenger.sol">IL1ScrollMessenger.sol</a>: L1 Scroll messenger interface
│   │   └── <a href="./src/L1/L1ScrollMessenger.sol">L1ScrollMessenger.sol</a>: L1 Scroll messenger contract
│   ├── <a href="./src/L2/">L2</a>: contracts on the L2
│   │   ├── <a href="./src/L2/gateways/">gateways</a>: Gateway router and individual gateway contracts
│   │   ├── <a href="./src/L2/predeploys/">predeploys</a>: Pre-deployed contracts on the L2 Scroll chain
│   │   ├── <a href="./src/L2/IL2ScrollMessenger.sol">IL2ScrollMessenger.sol</a>: L2 Scroll messenger interface
│   │   └── <a href="./src/L2/L2ScrollMessenger.sol">L2ScrollMessenger.sol</a>: L2 Scroll messenger contract
│   ├── <a href="./src/libraries/">libraries</a>: shared contract libraries
│   ├── <a href="./src/misc/">misc</a>: miscellaneous contracts
│   ├── <a href="./src/mocks/">mocks</a>: mock contracts used in the testing
│   └── <a href="./src/test/">test</a>: unit tests in solidity
├── <a href="./foundry.toml">foundry.toml</a>: configure foundry
├── <a href="./hardhat.config.ts">hardhat.config.ts</a>: configure hardhat
├── <a href="./remappings.txt">remappings.txt</a>: foundry dependency mappings
...
</pre>

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

- Run `git submodule update --init --recursive` to initialize git submodules.
- Run `yarn prettier:solidity` to run linting in fix mode, will auto-format all solidity codes.
- Run `yarn prettier` to run linting in fix mode, will auto-format all typescript codes.
- Run `yarn prepare` to install the precommit linting hook.
- Run `forge build` to compile contracts with foundry.
- Run `npx hardhat compile` to compile with hardhat.
- Run `forge test -vvv` to run foundry units tests. It will compile all contracts before running the unit tests.
- Run `npx hardhat test` to run integration tests. It may not compile all contracts before running, it's better to run `npx hardhat compile` first.
