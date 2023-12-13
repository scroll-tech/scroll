A library for interacting with Scroll contracts.

This library includes contracts and interfaces needed to interact with the Scroll Smart Contracts deployed on both Layer 1 and Layer 2. This includes deposting and withdrawing ETH, ERC20 tokens and NFTs or sending arbitrary messages.

# Overview

## Installations

```bash
npm install @scroll-tech/contracts
```

## Usage

Once installed, you can use the contracts in the library by importing them:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity 0.8.20;

import "@scroll-tech/contracts/L1/gateways/IL1ETHGateway.sol";

contract MyContract {
  function bridgeETH(address scrollBridge, uint256 gasLimit) public payable {
    IL1ETHGateway(scrollBridge).depositETH(msg.sender, msg.value, gasLimit);
  }
}

```

Visit the Bridge Documentation for API reference, architecture overview and guides with code examples.

# About Scroll

Scroll is a bytecode equivalent zkEVM for Ethereum. It enables native compatibility for existing Ethereum applications and tools. Learn more about Scroll [here](https://scroll.io/).
