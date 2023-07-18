// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {IERC721} from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import {IScrollERC721Extension} from "./IScrollERC721Extension.sol";

interface IScrollERC721 is IERC721, IScrollERC721Extension {}
