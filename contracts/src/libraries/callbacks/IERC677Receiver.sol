// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IERC677Receiver {
    function onTokenTransfer(
        address sender,
        uint256 value,
        bytes memory data
    ) external;
}
