// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

contract MockTarget {
    event ABC(uint256);

    function err() external pure {
        revert("test error");
    }

    function succeed() external {
        emit ABC(1);
    }
}
