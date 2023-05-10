// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {ScrollChain} from "../../L1/rollup/ScrollChain.sol";

contract MockScrollChain is ScrollChain {
    constructor() ScrollChain(0) {}

    /*
    function computePublicInputHash(uint64 accTotalL1Messages, Batch memory batch)
        external
        view
        returns (
            bytes32,
            uint64,
            uint64,
            uint64
        )
    {
        return _computePublicInputHash(accTotalL1Messages, batch);
    }
    */
}
