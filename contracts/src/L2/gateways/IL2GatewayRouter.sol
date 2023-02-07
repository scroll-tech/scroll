// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IScrollGateway } from "../../libraries/gateway/IScrollGateway.sol";
import { IL2ETHGateway } from "./IL2ETHGateway.sol";
import { IL2ERC20Gateway } from "./IL2ERC20Gateway.sol";

interface IL2GatewayRouter is IL2ETHGateway, IL2ERC20Gateway, IScrollGateway {}
