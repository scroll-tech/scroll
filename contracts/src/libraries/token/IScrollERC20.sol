// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IERC20Permit} from "@openzeppelin/contracts/token/ERC20/extensions/draft-IERC20Permit.sol";
import {IScrollERC20Extension} from "./IScrollERC20Extension.sol";

// The recommended ERC20 implementation for bridge token.
// deployed in L2 when original token is on L1
// deployed in L1 when original token is on L2
interface IScrollERC20 is IERC20, IERC20Permit, IScrollERC20Extension {

}
