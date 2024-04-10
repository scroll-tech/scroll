// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

import {IERC721} from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import {IScrollERC721Extension} from "./IScrollERC721Extension.sol";

// The recommended ERC721 implementation for bridge token.
// deployed in L2 when original token is on L1
// deployed in L1 when original token is on L2
interface IScrollERC721 is IERC721, IScrollERC721Extension {

}
