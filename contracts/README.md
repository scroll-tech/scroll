# Scroll Contracts

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

## Dependency

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
- Run `forge build` to compile contracts with foundry.
- Run `npx hardhat compile` to compile with hardhat.
- Run `forge test -vvv` to run foundry units tests. It will compile all contracts before running the unit tests.
- Run `npx hardhat test` to run integration tests. It may not compile all contracts before running, it's better to run `npx hardhat compile` first.

## TODO

- [ ] add proof verification codes
- [ ] layer1 to layer2 proof
- [ ] security analysis
