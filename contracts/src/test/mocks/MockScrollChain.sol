// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {ScrollChain} from "../../L1/rollup/ScrollChain.sol";

contract MockScrollChain is ScrollChain {
    constructor() ScrollChain(0) {}

    function setLastFinalizedBatchIndex(uint256 _lastFinalizedBatchIndex) external {
        lastFinalizedBatchIndex = _lastFinalizedBatchIndex;
    }
}
