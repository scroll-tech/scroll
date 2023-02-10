// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IL1ETHGateway } from "./IL1ETHGateway.sol";
import { IL1ERC20Gateway } from "./IL1ERC20Gateway.sol";

interface IL1GatewayRouter is IL1ETHGateway, IL1ERC20Gateway {}
