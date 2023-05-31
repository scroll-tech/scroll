// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

contract MockTarget {
    function err() external {
        revert("test error");
    }
}
