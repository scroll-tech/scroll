// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

contract MockTarget {
    event ABC(uint256);

    function err() pure external {
        revert("test error");
    }
    function succeed() external {
        emit ABC(1);
    }
}
