// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import {TransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import {MinimalForwarder} from "@openzeppelin/contracts/metatx/MinimalForwarder.sol";
